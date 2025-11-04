package http

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"go.uber.org/zap"

	"go-shorter/internal/delivery/middleware"
	"go-shorter/internal/domain"
	"go-shorter/internal/infra/queue"
	repoinfra "go-shorter/internal/infra/repository"
	ports "go-shorter/internal/repository"
	"go-shorter/internal/service/auth"
	"go-shorter/internal/service/ratelimit"
	"go-shorter/internal/service/slug"
)

type Server struct {
	Router      *chi.Mux
	Log         *zap.Logger
	JWT         *auth.JWTService
	Users       ports.UserRepository
	Links       ports.LinkRepository
	RateLimiter *ratelimit.RedisTokenBucket
	SlugGen     slug.Generator
	Enq         *queue.RedisListQueue
	Clicks      *repoinfra.ClicksAggRepoPG
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
	// recovery and logging
	r.Use(middleware.Recover(log))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	// request logging
	r.Use(middleware.RequestLogger(log))

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
	// enqueue OG crawl task: payload format slug|target_url
	if s.Enq != nil {
		_ = s.Enq.Enqueue(r.Context(), slugStr+"|"+link.TargetURL)
	}
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

func (s *Server) linkStats(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	q := r.URL.Query()
	var (
		from time.Time
		to   time.Time
	)
	bucket := q.Get("bucket")
	switch bucket {
	case "hour", "minute", "second":
		// ok
	default:
		bucket = "day"
	}
	fromStr := strings.TrimSpace(q.Get("from"))
	toStr := strings.TrimSpace(q.Get("to"))
	if fromStr != "" || toStr != "" {
		// both must be provided
		if fromStr == "" || toStr == "" {
			writeJSON(w, 400, response{Status: "error", Error: "from and to are required together (YYYY-MM-DD)"})
			return
		}
		var err error
		if strings.Contains(fromStr, "T") {
			from, err = time.Parse(time.RFC3339, fromStr)
		} else {
			from, err = time.Parse("2006-01-02", fromStr)
		}
		if err != nil {
			writeJSON(w, 400, response{Status: "error", Error: "invalid from date"})
			return
		}
		if strings.Contains(toStr, "T") {
			to, err = time.Parse(time.RFC3339, toStr)
		} else {
			to, err = time.Parse("2006-01-02", toStr)
		}
		if err != nil {
			writeJSON(w, 400, response{Status: "error", Error: "invalid to date"})
			return
		}
		from = from.UTC()
		to = to.UTC()
		if to.Before(from) {
			writeJSON(w, 400, response{Status: "error", Error: "to must be >= from"})
			return
		}
	} else {
		rangeStr := q.Get("range")
		if rangeStr == "" {
			rangeStr = "7d"
		}
		var dur time.Duration
		if strings.HasSuffix(rangeStr, "d") {
			daysStr := strings.TrimSuffix(rangeStr, "d")
			if n, err := strconv.Atoi(daysStr); err == nil && n > 0 {
				dur = time.Duration(n) * 24 * time.Hour
			}
		}
		if dur == 0 {
			if d, err := time.ParseDuration(rangeStr); err == nil {
				dur = d
			} else {
				dur = 7 * 24 * time.Hour
			}
		}
		to = time.Now().UTC().Truncate(24 * time.Hour)
		from = to.Add(-dur)
	}
	if s.Clicks == nil {
		writeJSON(w, 200, response{Status: "ok", Data: map[string]any{"points": []any{}, "total": 0}})
		return
	}
	points, err := s.Clicks.ListBySlugRange(r.Context(), 1, slug, from, to, bucket)
	if err != nil {
		writeJSON(w, 500, response{Status: "error", Error: err.Error()})
		return
	}
	var total int64
	out := make([]map[string]any, 0, len(points))
	for _, p := range points {
		total += p.Total
		if bucket == "day" {
			out = append(out, map[string]any{"day": p.Time.UTC().Format("2006-01-02"), "total": p.Total})
		} else {
			out = append(out, map[string]any{"day": p.Time.UTC().Format(time.RFC3339), "total": p.Total})
		}
	}
	writeJSON(w, 200, response{Status: "ok", Data: map[string]any{"points": out, "total": total}})
}

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
