package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/photo"
)

const photoColumns = "id, user_id, storage_key, url, ST_AsText(location), visibility, nsfw_status, taken_at, created_at"

type PhotoRepository struct {
	pool *pgxpool.Pool
}

func NewPhotoRepository(pool *pgxpool.Pool) *PhotoRepository {
	return &PhotoRepository{pool: pool}
}

func (r *PhotoRepository) GetByID(ctx context.Context, id string) (photo.Photo, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+photoColumns+" FROM photo WHERE id = $1", id)
	e, err := scanPhoto(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return photo.Photo{}, photo.ErrNotFound
	}
	return e, err
}

func (r *PhotoRepository) List(ctx context.Context) ([]photo.Photo, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+photoColumns+" FROM photo ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []photo.Photo
	for rows.Next() {
		e, err := scanPhoto(rows)
		if err != nil {
			return nil, err
		}
		entities = append(entities, e)
	}
	return entities, rows.Err()
}

func (r *PhotoRepository) Create(ctx context.Context, e photo.Photo) (photo.Photo, error) {
	row := r.pool.QueryRow(ctx,
		"INSERT INTO photo (user_id, storage_key, url, location, visibility, nsfw_status, taken_at) VALUES ($1, $2, $3, ST_GeomFromText($4, 4326), COALESCE(NULLIF($5, '')::visibility_level, 'public'), COALESCE(NULLIF($6, '')::nsfw_status, 'pending'), $7) RETURNING "+photoColumns,
		e.UserID, e.StorageKey, e.URL, wktArg(e.Location), e.Visibility, e.NSFWStatus, e.TakenAt)
	return scanPhoto(row)
}

func (r *PhotoRepository) Update(ctx context.Context, e photo.Photo) (photo.Photo, error) {
	row := r.pool.QueryRow(ctx,
		"UPDATE photo SET user_id = $2, storage_key = $3, url = $4, location = ST_GeomFromText($5, 4326), visibility = COALESCE(NULLIF($6, '')::visibility_level, visibility), nsfw_status = COALESCE(NULLIF($7, '')::nsfw_status, nsfw_status), taken_at = $8 WHERE id = $1 RETURNING "+photoColumns,
		e.ID, e.UserID, e.StorageKey, e.URL, wktArg(e.Location), e.Visibility, e.NSFWStatus, e.TakenAt)
	updated, err := scanPhoto(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return photo.Photo{}, photo.ErrNotFound
	}
	return updated, err
}

func (r *PhotoRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM photo WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return photo.ErrNotFound
	}
	return nil
}

func scanPhoto(row pgx.Row) (photo.Photo, error) {
	var e photo.Photo
	var geomLocation *string
	err := row.Scan(&e.ID, &e.UserID, &e.StorageKey, &e.URL, &geomLocation, &e.Visibility, &e.NSFWStatus, &e.TakenAt, &e.CreatedAt)
	if err != nil {
		return photo.Photo{}, err
	}
	e.Location = wktBytes(geomLocation)
	return e, nil
}
