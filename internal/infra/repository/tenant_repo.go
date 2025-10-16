package repository

import (
	"context"
	"go-shorter/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TenantRepoPG struct{ db *pgxpool.Pool }

func NewTenantRepoPG(db *pgxpool.Pool) *TenantRepoPG { return &TenantRepoPG{db: db} }

func (r *TenantRepoPG) CreateDefault(ctx context.Context, name string) (int64, error) {
	var id int64
	err := r.db.QueryRow(ctx, `INSERT INTO tenants(name, domain, status) VALUES($1, $1||'.local', 'active') RETURNING id`, name).Scan(&id)
	return id, err
}

func (r *TenantRepoPG) GetByName(ctx context.Context, name string) (*domain.Tenant, error) {
	row := r.db.QueryRow(ctx, `SELECT id, name, domain, status, created_at FROM tenants WHERE name=$1`, name)
	t := domain.Tenant{}
	if err := row.Scan(&t.ID, &t.Name, &t.Domain, &t.Status, &t.CreatedAt); err != nil { return nil, err }
	return &t, nil
}
