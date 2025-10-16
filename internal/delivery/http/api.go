package http

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"go.uber.org/zap"

	"go-shorter/internal/domain"
	"go-shorter/internal/repository"
	"go-shorter/internal/service/auth"
	"go-shorter/internal/service/ratelimit"
	"go-shorter/internal/service/slug"
)

type Server struct {
	Router      *chi.Mux
	Log         *zap.Logger
	JWT         *auth.JWTService
	Users       repository.UserRepository
	Links       repository.LinkRepository
	RateLimiter *ratelimit.RedisTokenBucket
	SlugGen     slug.Generator
}

type response struct {
	Status string `json:"status"`
	Data   any    `json:"data,omitempty"`
	Error  string `json:"error,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func NewServer(log *zap.Logger, jwt *auth.JWTService) *Server {
	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	s := &Server{Router: r, Log: log, JWT: jwt}
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	r.Route("/api/v1", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Post("/auth/login", s.handleLogin)
		})
		r.Group(func(r chi.Router) {
			r.Use(s.jwtMiddleware)
			r.Post("/links", s.createLink)
			r.Get("/links/{slug}", s.getLink)
			r.Patch("/links/{slug}", s.updateLink)
			r.Delete("/links/{slug}", s.deleteLink)
			r.Get("/links", s.listLinks)
			r.Get("/links/{slug}/stats", s.linkStats)
		})
	})
	return s
}

func (s *Server) jwtMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authz := r.Header.Get("Authorization")
		if !strings.HasPrefix(authz, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		tok := strings.TrimPrefix(authz, "Bearer ")
		if _, err := s.JWT.Verify(tok); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Handlers

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type createLinkReq struct {
	TargetURL  string     `json:"target_url"`
	CustomSlug *string    `json:"custom_slug"`
	ExpireAt   *time.Time `json:"expire_at"`
	MaxClicks  *int64     `json:"max_clicks"`
}

type linkDTO struct {
	Slug      string     `json:"slug"`
	TargetURL string     `json:"target_url"`
	IsActive  bool       `json:"is_active"`
	ExpireAt  *time.Time `json:"expire_at"`
	MaxClicks *int64     `json:"max_clicks"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func toDTO(l *domain.Link) linkDTO {
	return linkDTO{Slug: l.Slug, TargetURL: l.TargetURL, IsActive: l.IsActive, ExpireAt: l.ExpireAt, MaxClicks: l.MaxClicks, CreatedAt: l.CreatedAt, UpdatedAt: l.UpdatedAt}
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, response{Status: "error", Error: "email/password required"})
		return
	}
	if s.Users == nil {
		writeJSON(w, 500, response{Status: "error", Error: "auth not configured"})
		return
	}
	u, err := s.Users.GetByEmail(r.Context(), 1, req.Email)
	if err != nil || u == nil || !auth.VerifyPassword(u.PasswordHash, req.Password) {
		writeJSON(w, http.StatusUnauthorized, response{Status: "error", Error: "invalid credentials"})
		return
	}
	tok, _ := s.JWT.Sign(u.TenantID, u.ID, string(u.Role))
	writeJSON(w, 200, response{Status: "ok", Data: map[string]string{"access_token": tok}})
}

func (s *Server) createLink(w http.ResponseWriter, r *http.Request) {
	if s.RateLimiter != nil {
		ok, err := s.RateLimiter.Allow(r.Context(), "writetenant:default", 120)
		if err != nil || !ok {
			writeJSON(w, http.StatusTooManyRequests, response{Status: "error", Error: "rate limited"})
			return
		}
	}
	var req createLinkReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, response{Status: "error", Error: "bad json"})
		return
	}
	if !validURL(req.TargetURL) {
		writeJSON(w, 400, response{Status: "error", Error: "invalid target_url"})
		return
	}
	slugStr := ""
	if req.CustomSlug != nil && *req.CustomSlug != "" {
		slugStr = *req.CustomSlug
	} else {
		if s.SlugGen == nil {
			writeJSON(w, 500, response{Status: "error", Error: "slug generator not configured"})
			return
		}
		val, err := s.SlugGen.Next()
		if err != nil {
			writeJSON(w, 500, response{Status: "error", Error: "slug gen failed"})
			return
		}
		slugStr = val
	}
	link := &domain.Link{TenantID: 1, Slug: slugStr, TargetURL: req.TargetURL, IsActive: true, ExpireAt: req.ExpireAt, MaxClicks: req.MaxClicks, CreatedBy: 1}
	id, err := s.Links.Create(r.Context(), link)
	if err != nil {
		writeJSON(w, 500, response{Status: "error", Error: err.Error()})
		return
	}
	_ = id
	writeJSON(w, http.StatusCreated, response{Status: "ok", Data: toDTO(link)})
}

func (s *Server) getLink(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	l, err := s.Links.GetBySlug(r.Context(), 1, slug)
	if err != nil {
		writeJSON(w, 404, response{Status: "error", Error: "not found"})
		return
	}
	writeJSON(w, 200, response{Status: "ok", Data: toDTO(l)})
}

func (s *Server) updateLink(w http.ResponseWriter, r *http.Request) {
	slugStr := chi.URLParam(r, "slug")
	l, err := s.Links.GetBySlug(r.Context(), 1, slugStr)
	if err != nil {
		writeJSON(w, 404, response{Status: "error", Error: "not found"})
		return
	}
	var req createLinkReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, response{Status: "error", Error: "bad json"})
		return
	}
	if req.TargetURL != "" && !validURL(req.TargetURL) {
		writeJSON(w, 400, response{Status: "error", Error: "invalid target_url"})
		return
	}
	if req.TargetURL != "" {
		l.TargetURL = req.TargetURL
	}
	if req.ExpireAt != nil {
		l.ExpireAt = req.ExpireAt
	}
	if req.MaxClicks != nil {
		l.MaxClicks = req.MaxClicks
	}
	if err := s.Links.Update(r.Context(), l); err != nil {
		writeJSON(w, 500, response{Status: "error", Error: err.Error()})
		return
	}
	writeJSON(w, 200, response{Status: "ok", Data: toDTO(l)})
}

func (s *Server) deleteLink(w http.ResponseWriter, r *http.Request) {
	slugStr := chi.URLParam(r, "slug")
	if err := s.Links.Delete(r.Context(), 1, slugStr); err != nil {
		writeJSON(w, 500, response{Status: "error", Error: err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listLinks(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	items, err := s.Links.List(r.Context(), 1, q, 50, 0)
	if err != nil {
		writeJSON(w, 500, response{Status: "error", Error: err.Error()})
		return
	}
	out := make([]linkDTO, 0, len(items))
	for i := range items {
		out = append(out, toDTO(&items[i]))
	}
	writeJSON(w, 200, response{Status: "ok", Data: out})
}

func (s *Server) linkStats(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }

func validURL(u string) bool {
	parsed, err := url.Parse(u)
	if err != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	return parsed.Host != ""
}
