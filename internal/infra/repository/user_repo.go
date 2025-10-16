package repository

import (
	"context"
	"go-shorter/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepoPG struct{ db *pgxpool.Pool }

func NewUserRepoPG(db *pgxpool.Pool) *UserRepoPG { return &UserRepoPG{db: db} }

func (r *UserRepoPG) CreateAdmin(ctx context.Context, tenantID int64, email, passwordHash string) (int64, error) {
	var id int64
	err := r.db.QueryRow(ctx, `INSERT INTO users(tenant_id,email,password_hash,role,status) VALUES($1,$2,$3,'admin','active') RETURNING id`, tenantID, email, passwordHash).Scan(&id)
	return id, err
}

func (r *UserRepoPG) GetByEmail(ctx context.Context, tenantID int64, email string) (*domain.User, error) {
	row := r.db.QueryRow(ctx, `SELECT id, tenant_id, email, password_hash, role, status, created_at, updated_at FROM users WHERE tenant_id=$1 AND email=$2`, tenantID, email)
	u := domain.User{}
	if err := row.Scan(&u.ID, &u.TenantID, &u.Email, &u.PasswordHash, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt); err != nil { return nil, err }
	return &u, nil
}
