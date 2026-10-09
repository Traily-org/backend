package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/publication"
)

type PublicationRepository struct{ pool *pgxpool.Pool }

func NewPublicationRepository(pool *pgxpool.Pool) *PublicationRepository {
	return &PublicationRepository{pool: pool}
}
func (r *PublicationRepository) GetByID(ctx context.Context, id string) (publication.Publication, error) {
	row := r.pool.QueryRow(ctx, "SELECT id, user_id, track_id, trace_id, title, content, published_at, created_at, updated_at FROM publication WHERE id = $1", id)
	var p publication.Publication
	err := row.Scan(&p.ID, &p.UserID, &p.TrackID, &p.TraceID, &p.Title, &p.Content, &p.PublishedAt, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return publication.Publication{}, publication.ErrNotFound
	}
	return p, err
}
func (r *PublicationRepository) List(ctx context.Context) ([]publication.Publication, error) {
	rows, err := r.pool.Query(ctx, "SELECT id, user_id, track_id, trace_id, title, content, published_at, created_at, updated_at FROM publication ORDER BY published_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pubs []publication.Publication
	for rows.Next() {
		var p publication.Publication
		if err := rows.Scan(&p.ID, &p.UserID, &p.TrackID, &p.TraceID, &p.Title, &p.Content, &p.PublishedAt, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		pubs = append(pubs, p)
	}
	return pubs, rows.Err()
}
func (r *PublicationRepository) Create(ctx context.Context, p publication.Publication) (publication.Publication, error) {
	row := r.pool.QueryRow(ctx, "INSERT INTO publication (user_id, track_id, trace_id, title, content) VALUES ($1, $2, $3, $4, $5) RETURNING id, user_id, track_id, trace_id, title, content, published_at, created_at, updated_at", p.UserID, p.TrackID, p.TraceID, p.Title, p.Content)
	err := row.Scan(&p.ID, &p.UserID, &p.TrackID, &p.TraceID, &p.Title, &p.Content, &p.PublishedAt, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}
func (r *PublicationRepository) Update(ctx context.Context, p publication.Publication) (publication.Publication, error) {
	row := r.pool.QueryRow(ctx, "UPDATE publication SET user_id=$2, track_id=$3, trace_id=$4, title=$5, content=$6, updated_at=NOW() WHERE id=$1 RETURNING id, user_id, track_id, trace_id, title, content, published_at, created_at, updated_at", p.ID, p.UserID, p.TrackID, p.TraceID, p.Title, p.Content)
	err := row.Scan(&p.ID, &p.UserID, &p.TrackID, &p.TraceID, &p.Title, &p.Content, &p.PublishedAt, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return publication.Publication{}, publication.ErrNotFound
	}
	return p, err
}
func (r *PublicationRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM publication WHERE id=$1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return publication.ErrNotFound
	}
	return nil
}
