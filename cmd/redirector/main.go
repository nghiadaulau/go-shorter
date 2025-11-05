package main

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

	"go-shorter/internal/delivery/middleware"
	"go-shorter/internal/infra/postgres"
	infredis "go-shorter/internal/infra/redis"
	repo "go-shorter/internal/infra/repository"
	"go-shorter/internal/service/cache"
	"go-shorter/pkg/shared/config"
	"go-shorter/pkg/shared/logger"
)

var cfgPath string
var Version string

var (
	cacheHitRatio = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "redirect_cache_hits_total",
		Help: "Redirect cache hits",
	}, []string{"result"})
	redirectLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "redirect_latency_seconds",
		Help:    "Latency of redirect handler",
		Buckets: prometheus.DefBuckets,
	})
)

var ogTpl = template.Must(template.New("og").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta property="og:title" content="{{.Title}}" />
  <meta property="og:description" content="{{.Description}}" />
  {{if .Image}}<meta property="og:image" content="{{.Image}}" />{{end}}
  <meta property="og:url" content="{{.ShortURL}}" />
  <meta http-equiv="refresh" content="0;url={{.TargetURL}}" />
  <link rel="canonical" href="{{.TargetURL}}" />
</head>
<body>Redirecting...</body>
</html>`))

func isBot(ua string) bool {
	ua = strings.ToLower(ua)
	bots := []string{"facebookexternalhit", "twitterbot", "slackbot", "whatsapp", "discordbot", "telegrambot", "googlebot", "bingbot"}
	for _, b := range bots {
		if strings.Contains(ua, b) {
			return true
		}
	}
	return false
}

func main() {
	prometheus.MustRegister(cacheHitRatio, redirectLatency)
	cmd := &cobra.Command{
		Use:   "redirector",
		Short: "Redirect edge server",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, log, err := loadConfig()
			if err != nil {
				return err
			}
			defer log.Sync()
			ctx := context.Background()
			if Version != "" {
				log.Info("version", zap.String("version", Version))
			}
			pool, err := postgres.NewPool(ctx, cfg.DB.DSN, cfg.DB.MaxOpenConns, cfg.DB.MaxIdleConns)
			if err != nil {
				return err
			}
			defer pool.Close()
			redis := infredis.NewClient(cfg.Redis.Addr, cfg.Redis.DB, cfg.Redis.Pool)
			links := repo.NewLinkRepoPG(pool)
			linkMeta := repo.NewLinkMetaRepoPG(pool)

			// in-memory hot cache and singleflight group
			mem := cache.NewTTL(5 * time.Minute)
			var sf singleflight.Group

			r := chi.NewRouter()
			// middlewares MUST come before any routes
			r.Use(middleware.Recover(log))
			r.Use(middleware.RequestLogger(log))
			r.Handle("/metrics", promhttp.Handler())
			r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
			r.Get("/{slug}", func(w http.ResponseWriter, r *http.Request) {
				start := time.Now()
				slug := chi.URLParam(r, "slug")
				key := fmt.Sprintf("sl:%s:%s", "default", slug)
				ctx := r.Context()
				ua := r.Header.Get("User-Agent")

				if v, ok := mem.Get(key); ok && v != "" {
					cacheHitRatio.WithLabelValues("hit").Inc()
					go func(sl string) {
						_ = redis.LPush(context.Background(), "clicks:agg", time.Now().UTC().Format(time.RFC3339)+"|"+sl+"|1|1").Err()
					}(slug)
					if isBot(ua) {
						serveOG(w, r, slug, v, links, linkMeta)
						redirectLatency.Observe(time.Since(start).Seconds())
						return
					}
					http.Redirect(w, r, v, http.StatusFound)
					redirectLatency.Observe(time.Since(start).Seconds())
					return
				}

				ctxR, cancelR := context.WithTimeout(ctx, 300*time.Millisecond)
				if target, err := redis.Get(ctxR, key).Result(); err == nil && target != "" {
					cancelR()
					mem.Set(key, target)
					cacheHitRatio.WithLabelValues("hit").Inc()
					go func(sl string) {
						_ = redis.LPush(context.Background(), "clicks:agg", time.Now().UTC().Format(time.RFC3339)+"|"+sl+"|1|1").Err()
					}(slug)
					if isBot(ua) {
						serveOG(w, r, slug, target, links, linkMeta)
						redirectLatency.Observe(time.Since(start).Seconds())
						return
					}
					http.Redirect(w, r, target, http.StatusFound)
					redirectLatency.Observe(time.Since(start).Seconds())
					return
				}
				cancelR()
				cacheHitRatio.WithLabelValues("miss").Inc()

				v, _, _ := sf.Do(key, func() (interface{}, error) {
					ctxDB, cancelDB := context.WithTimeout(ctx, 800*time.Millisecond)
					defer cancelDB()
					l, err := links.GetBySlug(ctxDB, 1, slug)
					if err != nil || l == nil || !l.IsActive {
						log.Warn("db miss", zap.String("slug", slug), zap.Error(err))
						return "", fmt.Errorf("notfound")
					}
					if err := redis.Set(ctx, key, l.TargetURL, 24*time.Hour).Err(); err != nil {
						log.Warn("redis set failed", zap.Error(err))
					}
					mem.Set(key, l.TargetURL)
					return l.TargetURL, nil
				})
				target, _ := v.(string)
				if target == "" {
					w.WriteHeader(http.StatusNotFound)
					redirectLatency.Observe(time.Since(start).Seconds())
					return
				}
				go func(sl string) {
					_ = redis.LPush(context.Background(), "clicks:agg", time.Now().UTC().Format(time.RFC3339)+"|"+sl+"|1|1").Err()
				}(slug)
				if isBot(ua) {
					serveOG(w, r, slug, target, links, linkMeta)
					redirectLatency.Observe(time.Since(start).Seconds())
					return
				}
				http.Redirect(w, r, target, http.StatusFound)
				redirectLatency.Observe(time.Since(start).Seconds())
			})
			addr := fmt.Sprintf(":%d", 8085)
			log.Info("redirector starting", zap.String("addr", addr))
			return http.ListenAndServe(addr, r)
		},
	}
	cmd.PersistentFlags().StringVar(&cfgPath, "config", "", "path to config file")
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func serveOG(w http.ResponseWriter, r *http.Request, slug, target string, links *repo.LinkRepoPG, metaRepo *repo.LinkMetaRepoPG) {
	// Try to load metadata from DB for this slug
	title := slug
	desc := target
	image := ""
	// use a short context to avoid blocking bots too long
	ctx, cancel := context.WithTimeout(r.Context(), 300*time.Millisecond)
	defer cancel()
	if l, err := links.GetBySlug(ctx, 1, slug); err == nil && l != nil {
		if m, err2 := metaRepo.GetByLinkID(ctx, l.ID); err2 == nil && m != nil && !m.NoPreview {
			if m.Title != "" {
				title = m.Title
			}
			if m.Description != "" {
				desc = m.Description
			}
			if m.OGImage != "" {
				image = m.OGImage
			}
		}
	}
	shortURL := ""
	if host := r.Host; host != "" {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		shortURL = fmt.Sprintf("%s://%s/%s", scheme, host, slug)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := struct {
		Title       string
		Description string
		Image       string
		ShortURL    string
		TargetURL   string
	}{
		Title:       title,
		Description: desc,
		Image:       image,
		ShortURL:    shortURL,
		TargetURL:   target,
	}
	_ = ogTpl.Execute(w, data)
}

func loadConfig() (*config.AppConfig, *zap.Logger, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	if cfgPath != "" {
		v.SetConfigFile(cfgPath)
	} else {
		v.SetConfigName("config")
		v.AddConfigPath("./config")
	}
	v.SetEnvPrefix("APP")
	v.AutomaticEnv()
	_ = v.ReadInConfig()
	var cfg config.AppConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, nil, err
	}
	log, err := logger.New()
	if err != nil {
		return nil, nil, err
	}
	return &cfg, log, nil
}
