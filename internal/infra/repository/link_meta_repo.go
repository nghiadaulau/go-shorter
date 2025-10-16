package repository

import (
	"context"
	"go-shorter/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LinkMetaRepoPG struct{ db *pgxpool.Pool }

func NewLinkMetaRepoPG(db *pgxpool.Pool) *LinkMetaRepoPG { return &LinkMetaRepoPG{db: db} }

func (r *LinkMetaRepoPG) GetByLinkID(ctx context.Context, linkID int64) (*domain.LinkMeta, error) {
	row := r.db.QueryRow(ctx, `SELECT link_id, title, description, og_image, no_preview FROM link_meta WHERE link_id=$1`, linkID)
	m := domain.LinkMeta{}
	if err := row.Scan(&m.LinkID, &m.Title, &m.Description, &m.OGImage, &m.NoPreview); err != nil { return nil, err }
	return &m, nil
}
