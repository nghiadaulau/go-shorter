package repository

import (
	"context"
	"go-shorter/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LinkRepoPG struct{ db *pgxpool.Pool }

func NewLinkRepoPG(db *pgxpool.Pool) *LinkRepoPG { return &LinkRepoPG{db: db} }

func (r *LinkRepoPG) Create(ctx context.Context, link *domain.Link) (int64, error) {
	var id int64
	err := r.db.QueryRow(ctx, `INSERT INTO links(tenant_id, slug, target_url, is_active, expire_at, max_clicks, created_by) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`, link.TenantID, link.Slug, link.TargetURL, link.IsActive, link.ExpireAt, link.MaxClicks, link.CreatedBy).Scan(&id)
	return id, err
}

func (r *LinkRepoPG) GetBySlug(ctx context.Context, tenantID int64, slug string) (*domain.Link, error) {
	row := r.db.QueryRow(ctx, `SELECT id, tenant_id, slug, target_url, is_active, expire_at, max_clicks, created_by, created_at, updated_at FROM links WHERE tenant_id=$1 AND slug=$2`, tenantID, slug)
	l := domain.Link{}
	if err := row.Scan(&l.ID, &l.TenantID, &l.Slug, &l.TargetURL, &l.IsActive, &l.ExpireAt, &l.MaxClicks, &l.CreatedBy, &l.CreatedAt, &l.UpdatedAt); err != nil { return nil, err }
	return &l, nil
}

func (r *LinkRepoPG) Update(ctx context.Context, link *domain.Link) error {
	_, err := r.db.Exec(ctx, `UPDATE links SET target_url=$1, is_active=$2, expire_at=$3, max_clicks=$4, updated_at=now() WHERE tenant_id=$5 AND slug=$6`, link.TargetURL, link.IsActive, link.ExpireAt, link.MaxClicks, link.TenantID, link.Slug)
	return err
}

func (r *LinkRepoPG) Delete(ctx context.Context, tenantID int64, slug string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM links WHERE tenant_id=$1 AND slug=$2`, tenantID, slug)
	return err
}

func (r *LinkRepoPG) List(ctx context.Context, tenantID int64, q string, limit, offset int) ([]domain.Link, error) {
	rows, err := r.db.Query(ctx, `SELECT id, tenant_id, slug, target_url, is_active, expire_at, max_clicks, created_by, created_at, updated_at FROM links WHERE tenant_id=$1 AND ($2='' OR slug ILIKE '%'||$2||'%' OR target_url ILIKE '%'||$2||'%') ORDER BY id DESC LIMIT $3 OFFSET $4`, tenantID, q, limit, offset)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []domain.Link
	for rows.Next() {
		var l domain.Link
		if err := rows.Scan(&l.ID, &l.TenantID, &l.Slug, &l.TargetURL, &l.IsActive, &l.ExpireAt, &l.MaxClicks, &l.CreatedBy, &l.CreatedAt, &l.UpdatedAt); err != nil { return nil, err }
		out = append(out, l)
	}
	return out, rows.Err()
}
