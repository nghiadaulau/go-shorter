package repository

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	
)

type LinkMetaRepoPG struct{ db *pgxpool.Pool }

func NewLinkMetaRepoPG(db *pgxpool.Pool) *LinkMetaRepoPG { return &LinkMetaRepoPG{db: db} }

func (r *LinkMetaRepoPG) GetByLinkID(ctx context.Context, linkID int64) (*domain.LinkMeta, error) {
	row := r.db.QueryRow(ctx, `SELECT link_id, title, description, og_image, no_preview FROM link_meta WHERE link_id=$1`, linkID)
	m := domain.LinkMeta{}
	if err := row.Scan(&m.LinkID, &m.Title, &m.Description, &m.OGImage, &m.NoPreview); err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *LinkMetaRepoPG) Upsert(ctx context.Context, m *domain.LinkMeta) error {
	_, err := r.db.Exec(ctx, `INSERT INTO link_meta(link_id,title,description,og_image,no_preview)
	VALUES($1,$2,$3,$4,COALESCE($5,false))
	ON CONFLICT (link_id) DO UPDATE SET title=EXCLUDED.title, description=EXCLUDED.description, og_image=EXCLUDED.og_image, no_preview=EXCLUDED.no_preview`,
		m.LinkID, m.Title, m.Description, m.OGImage, m.NoPreview)
	return err
}
