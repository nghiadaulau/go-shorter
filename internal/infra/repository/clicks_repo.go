package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ClicksAggRepoPG struct{ db *pgxpool.Pool }

type ClickPoint struct {
	Time  time.Time
	Total int64
}

func NewClicksAggRepoPG(db *pgxpool.Pool) *ClicksAggRepoPG { return &ClicksAggRepoPG{db: db} }

func (r *ClicksAggRepoPG) Inc(ctx context.Context, ts time.Time, slug string, tenantID int64, delta int64) error {
	// store by second bucket to allow higher resolution aggregations later
	_, err := r.db.Exec(ctx, `INSERT INTO clicks_agg(bucket, slug, tenant_id, total)
    VALUES(date_trunc('second', $1::timestamptz), $2, $3, $4)
    ON CONFLICT (bucket, slug, tenant_id) DO UPDATE SET total = clicks_agg.total + EXCLUDED.total`, ts.UTC(), slug, tenantID, delta)
	return err
}

func (r *ClicksAggRepoPG) ListBySlugRange(ctx context.Context, tenantID int64, slug string, from, to time.Time, bucket string) ([]ClickPoint, error) {
	agg := "day"
	switch bucket {
	case "hour":
		agg = "hour"
	case "minute":
		agg = "minute"
	case "second":
		agg = "second"
	}
	q := `SELECT date_trunc('` + agg + `', bucket) AS ts, SUM(total) FROM clicks_agg
          WHERE tenant_id=$1 AND slug=$2 AND bucket BETWEEN $3 AND $4
          GROUP BY ts ORDER BY ts ASC`
	rows, err := r.db.Query(ctx, q, tenantID, slug, from.UTC(), to.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ClickPoint
	for rows.Next() {
		var p ClickPoint
		if err := rows.Scan(&p.Time, &p.Total); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
