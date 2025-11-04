package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, dsn string, maxOpen, maxIdle int) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = int32(maxOpen)
	cfg.MinConns = int32(maxIdle)
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 15 * time.Minute
	cfg.HealthCheckPeriod = 30 * time.Second
	cfg.ConnConfig.ConnectTimeout = 5 * time.Second
	if cfg.ConnConfig.Config.RuntimeParams == nil {
		cfg.ConnConfig.Config.RuntimeParams = map[string]string{}
	}
	cfg.ConnConfig.Config.RuntimeParams["statement_timeout"] = "3000" // 3s
	return pgxpool.NewWithConfig(ctx, cfg)
}
