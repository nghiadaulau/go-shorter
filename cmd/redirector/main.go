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

	"go-shorter/internal/infra/postgres"
	infredis "go-shorter/internal/infra/redis"
	repo "go-shorter/internal/infra/repository"
	"go-shorter/pkg/shared/config"
	"go-shorter/pkg/shared/logger"
)

var cfgPath string

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
	for _, b := range bots { if strings.Contains(ua, b) { return true } }
	return false
}

func main() {
	prometheus.MustRegister(cacheHitRatio, redirectLatency)
	cmd := &cobra.Command{
		Use:   "redirector",
		Short: "Redirect edge server",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, log, err := loadConfig()
			if err != nil { return err }
			defer log.Sync()
			ctx := context.Background()
			pool, err := postgres.NewPool(ctx, cfg.DB.DSN, cfg.DB.MaxOpenConns, cfg.DB.MaxIdleConns)
			if err != nil { return err }
			defer pool.Close()
			redis := infredis.NewClient(cfg.Redis.Addr, cfg.Redis.DB, cfg.Redis.Pool)
			links := repo.NewLinkRepoPG(pool)
			linkMeta := repo.NewLinkMetaRepoPG(pool)

			r := chi.NewRouter()
			r.Handle("/metrics", promhttp.Handler())
			r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
			r.Get("/{slug}", func(w http.ResponseWriter, r *http.Request) {
				start := time.Now()
				slug := chi.URLParam(r, "slug")
				key := fmt.Sprintf("sl:%s:%s", "default", slug)
				ctx := r.Context()
				ua := r.Header.Get("User-Agent")

				// short deadline for redis
				ctxR, cancelR := context.WithTimeout(ctx, 120*time.Millisecond)
				defer cancelR()
				if target, err := redis.Get(ctxR, key).Result(); err == nil && target != "" {
					cacheHitRatio.WithLabelValues("hit").Inc()
					if isBot(ua) {
						serveOG(w, slug, target, linkMeta, redis)
						redirectLatency.Observe(time.Since(start).Seconds())
						return
					}
					http.Redirect(w, r, target, http.StatusFound)
					redirectLatency.Observe(time.Since(start).Seconds())
					return
				}
				cacheHitRatio.WithLabelValues("miss").Inc()
				// short deadline for DB
				ctxDB, cancelDB := context.WithTimeout(ctx, 200*time.Millisecond)
				defer cancelDB()
				l, err := links.GetBySlug(ctxDB, 1, slug)
				if err != nil || !l.IsActive {
					w.WriteHeader(http.StatusNotFound)
					redirectLatency.Observe(time.Since(start).Seconds())
					return
				}
				_ = redis.Set(ctx, key, l.TargetURL, 24*time.Hour).Err()
				if isBot(ua) {
					serveOG(w, slug, l.TargetURL, linkMeta, redis)
					redirectLatency.Observe(time.Since(start).Seconds())
					return
				}
				http.Redirect(w, r, l.TargetURL, http.StatusFound)
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

func serveOG(w http.ResponseWriter, slug, target string, metaRepo *repo.LinkMetaRepoPG, redisClient *infredis.Client) {
	// best-effort load meta (no strict timeout here for brevity)
	// render minimal OG if not found
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := struct {
		Title     string
		Description string
		Image     string
		ShortURL  string
		TargetURL string
	}{
		Title:     slug,
		Description: target,
		ShortURL:  "",
		TargetURL: target,
	}
	_ = ogTpl.Execute(w, data)
}

func loadConfig() (*config.AppConfig, *zap.Logger, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	if cfgPath != "" {
		v.SetConfigFile(cfgPath)
	} else {
		v.SetConfigName("config.example")
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
