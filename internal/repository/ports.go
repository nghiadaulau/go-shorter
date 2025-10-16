package repository

import (
	"context"
	"go-shorter/internal/domain"
)

type TenantRepository interface {
	CreateDefault(ctx context.Context, name string) (int64, error)
	GetByName(ctx context.Context, name string) (*domain.Tenant, error)
}

type UserRepository interface {
	CreateAdmin(ctx context.Context, tenantID int64, email, passwordHash string) (int64, error)
	GetByEmail(ctx context.Context, tenantID int64, email string) (*domain.User, error)
}

type LinkRepository interface {
	Create(ctx context.Context, link *domain.Link) (int64, error)
	GetBySlug(ctx context.Context, tenantID int64, slug string) (*domain.Link, error)
	Update(ctx context.Context, link *domain.Link) error
	Delete(ctx context.Context, tenantID int64, slug string) error
	List(ctx context.Context, tenantID int64, q string, limit, offset int) ([]domain.Link, error)
}

type LinkMetaRepository interface {
	GetByLinkID(ctx context.Context, linkID int64) (*domain.LinkMeta, error)
}
