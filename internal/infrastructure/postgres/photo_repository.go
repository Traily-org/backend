package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/photo"
)

const photoColumns = "id, user_id, storage_key, url, ST_AsText(location), visibility, nsfw_status, taken_at, created_at"

type PhotoRepository struct{ pool *pgxpool.Pool }

func NewPhotoRepository(pool *pgxpool.Pool) *PhotoRepository {
	return &PhotoRepository{pool: pool}
}

func scanPhoto(row pgx.Row) (photo.Photo, error) {
	var p photo.Photo
	var loc *string
	err := row.Scan(&p.ID, &p.UserID, &p.StorageKey, &p.URL, &loc, &p.Visibility, &p.NSFWStatus, &p.TakenAt, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return photo.Photo{}, photo.ErrNotFound
	}
	if err == nil && loc != nil {
		p.Location = []byte(*loc)
	}
	return p, err
}

func nullableWKT(b []byte) *string {
	if b == nil {
		return nil
	}
	s := string(b)
	return &s
}

func (r *PhotoRepository) GetByID(ctx context.Context, id string) (photo.Photo, error) {
	return scanPhoto(r.pool.QueryRow(ctx, "SELECT "+photoColumns+" FROM photos WHERE id = $1", id))
}

func (r *PhotoRepository) List(ctx context.Context) ([]photo.Photo, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+photoColumns+" FROM photos ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []photo.Photo
	for rows.Next() {
		p, err := scanPhoto(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func (r *PhotoRepository) Create(ctx context.Context, p photo.Photo) (photo.Photo, error) {
	return scanPhoto(r.pool.QueryRow(ctx, "INSERT INTO photos (user_id, storage_key, url, location, visibility, nsfw_status, taken_at) VALUES ($1, $2, $3, CASE WHEN $4::text IS NOT NULL THEN ST_GeomFromText($4, 4326) END, $5, $6, $7) RETURNING "+photoColumns,
		p.UserID, p.StorageKey, p.URL, nullableWKT(p.Location), p.Visibility, p.NSFWStatus, p.TakenAt))
}

func (r *PhotoRepository) Update(ctx context.Context, p photo.Photo) (photo.Photo, error) {
	return scanPhoto(r.pool.QueryRow(ctx, "UPDATE photos SET user_id=$2, storage_key=$3, url=$4, location=CASE WHEN $5::text IS NOT NULL THEN ST_GeomFromText($5, 4326) END, visibility=$6, nsfw_status=$7, taken_at=$8 WHERE id=$1 RETURNING "+photoColumns,
		p.ID, p.UserID, p.StorageKey, p.URL, nullableWKT(p.Location), p.Visibility, p.NSFWStatus, p.TakenAt))
}

func (r *PhotoRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM photos WHERE id=$1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return photo.ErrNotFound
	}
	return nil
}
