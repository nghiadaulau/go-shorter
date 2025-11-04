package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"go-shorter/internal/domain"
	"go-shorter/internal/infra/postgres"
	"go-shorter/internal/infra/queue"
	infredis "go-shorter/internal/infra/redis"
	repo "go-shorter/internal/infra/repository"
	"go-shorter/internal/service/ogcrawl"
	"go-shorter/pkg/shared/config"
	"go-shorter/pkg/shared/logger"
	"go.uber.org/zap"
)

var cfgPath string

func main() {
	cmd := &cobra.Command{
		Use:   "click-collector",
		Short: "Collects click events / OG crawler",
		RunE: func(cmd *cobra.Command, args []string) error {
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
			redis := infredis.NewClient(cfg.Redis.Addr, cfg.Redis.DB, cfg.Redis.Pool)
			q := queue.NewRedisListQueue(redis, "og:crawl")
			links := repo.NewLinkRepoPG(pool)
			metas := repo.NewLinkMetaRepoPG(pool)

			for {
				payload, err := q.Dequeue(ctx, 10*time.Second)
				if err != nil || payload == "" {
					continue
				}
				parts := strings.SplitN(payload, "|", 2)
				if len(parts) != 2 {
					continue
				}
				slug, target := parts[0], parts[1]
				ctxFetch, cancel := context.WithTimeout(ctx, 8*time.Second)
				meta, err := ogcrawl.Fetch(ctxFetch, target)
				cancel()
				if err != nil {
					log.Warn("og fetch failed", zap.String("slug", slug), zap.Error(err))
					continue
				}

				var l *domain.Link
				var lastErr error
				for i := 0; i < 3; i++ {
					ctxDB, cancelDB := context.WithTimeout(ctx, 6*time.Second)
					l, lastErr = links.GetBySlug(ctxDB, 1, slug)
					cancelDB()
					if lastErr == nil && l != nil {
						break
					}
					time.Sleep(250 * time.Millisecond)
				}
				if lastErr != nil || l == nil {
					log.Warn("get by slug failed", zap.String("slug", slug), zap.Error(lastErr))
					continue
				}

				m := &domain.LinkMeta{LinkID: l.ID, Title: meta.Title, Description: meta.Description, OGImage: meta.Image, NoPreview: false}
				if m.Title == "" {
					m.Title = slug
				}
				if m.Description == "" {
					m.Description = target
				}
				if err := metas.Upsert(ctx, m); err != nil {
					log.Warn("meta upsert failed", zap.Int64("link_id", l.ID), zap.Error(err))
				}
			}
		},
	}
	cmd.PersistentFlags().StringVar(&cfgPath, "config", "", "path to config file")
	if err := cmd.Execute(); err != nil {
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
