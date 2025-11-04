package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	apihttp "go-shorter/internal/delivery/http"
	"go-shorter/internal/domain"
	"go-shorter/internal/infra/postgres"
	"go-shorter/internal/infra/queue"
	infredis "go-shorter/internal/infra/redis"
	repoinfra "go-shorter/internal/infra/repository"
	"go-shorter/internal/service/auth"
	"go-shorter/internal/service/ratelimit"
	"go-shorter/internal/service/slug"
	"go-shorter/pkg/shared/config"
	"go-shorter/pkg/shared/logger"
	"go-shorter/pkg/shared/telemetry"
)

var (
	cfgPath string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "shortlink-api",
		Short: "Shortlink API server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer()
		},
	}
	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", "", "path to config file")

	seedCmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed default data",
		RunE:  runSeed,
	}
	seedCmd.Flags().String("admin-email", "", "admin email")
	seedCmd.Flags().String("password", "", "admin password")
	seedCmd.Flags().String("tenant", "default", "tenant name")
	_ = viper.BindPFlag("seed.adminEmail", seedCmd.Flags().Lookup("admin-email"))
	_ = viper.BindPFlag("seed.password", seedCmd.Flags().Lookup("password"))
	_ = viper.BindPFlag("seed.tenant", seedCmd.Flags().Lookup("tenant"))

	rootCmd.AddCommand(seedCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
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

	if err := v.ReadInConfig(); err != nil {
		// Proceed with env-only config
	}

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

func runServer() error {
	cfg, log, err := loadConfig()
	if err != nil {
		return err
	}
	defer log.Sync()

	ctx := context.Background()
	// telemetry
	var shutdown func(context.Context) error
	if cfg.Telemetry.Enable && cfg.Telemetry.OTLPEndpoint != "" {
		shutdown, err = telemetry.Setup(ctx, telemetry.Options{Endpoint: cfg.Telemetry.OTLPEndpoint, Service: "shortlink-api"})
		if err != nil {
			log.Warn("otel setup failed", zap.Error(err))
		}
	}
	defer func() {
		if shutdown != nil {
			_ = shutdown(context.Background())
		}
	}()

	// deps
	pool, err := postgres.NewPool(ctx, cfg.DB.DSN, cfg.DB.MaxOpenConns, cfg.DB.MaxIdleConns)
	if err != nil {
		return err
	}
	defer pool.Close()
	redisClient := infredis.NewClient(cfg.Redis.Addr, cfg.Redis.DB, cfg.Redis.Pool)

	jwtSvc := auth.NewJWT(cfg.JWT.Secret, cfg.JWT.Issuer, cfg.JWT.TTL)

	s := apihttp.NewServer(log, jwtSvc)
	// wire dependencies
	s.Links = repoinfra.NewLinkRepoPG(pool)
	s.Users = repoinfra.NewUserRepoPG(pool)
	s.RateLimiter = ratelimit.NewRedisTokenBucket(redisClient)
	s.SlugGen = slug.NewBase62(pool, "slug_seq")
	// OG crawl queue key
	s.Enq = queue.NewRedisListQueue(redisClient, "og:crawl")
	s.Clicks = repoinfra.NewClicksAggRepoPG(pool)

	// prometheus metrics endpoint
	s.Router.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:           s.Router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Info("api server starting", zap.Int("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	// graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctxSh, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(ctxSh)
}

func runSeed(cmd *cobra.Command, args []string) error {
	cfg, log, err := loadConfig()
	if err != nil {
		return err
	}
	defer log.Sync()
	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, cfg.DB.DSN, cfg.DB.MaxOpenConns, cfg.DB.MaxIdleConns)
	if err != nil {
		return err
	}
	defer pool.Close()

	tenantName := viper.GetString("seed.tenant")
	email := viper.GetString("seed.adminEmail")
	password := viper.GetString("seed.password")
	if tenantName == "" || email == "" || password == "" {
		return fmt.Errorf("missing seed flags")
	}

	tenantRepo := repoinfra.NewTenantRepoPG(pool)
	userRepo := repoinfra.NewUserRepoPG(pool)

	ten, err := tenantRepo.GetByName(ctx, tenantName)
	if err != nil || ten == nil {
		id, err2 := tenantRepo.CreateDefault(ctx, tenantName)
		if err2 != nil {
			return err2
		}
		ten = &domain.Tenant{ID: id, Name: tenantName}
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	_, err = userRepo.CreateAdmin(ctx, ten.ID, email, hash)
	if err != nil {
		return err
	}
	log.Info("seeded admin user", zap.String("email", email), zap.Int64("tenant_id", ten.ID))
	return nil
}
