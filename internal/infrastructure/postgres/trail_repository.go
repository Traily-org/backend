package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/trail"
)

const trailColumns = "id, user_id, name, description, difficulty, distance_m, elevation_gain_m, elevation_loss_m, estimated_duration_s, ST_AsText(path), is_official, is_published, created_at, updated_at"

type TrailRepository struct {
	pool *pgxpool.Pool
}

func NewTrailRepository(pool *pgxpool.Pool) *TrailRepository {
	return &TrailRepository{pool: pool}
}

func (r *TrailRepository) GetByID(ctx context.Context, id string) (trail.Trail, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+trailColumns+" FROM trail WHERE id = $1", id)
	e, err := scanTrail(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return trail.Trail{}, trail.ErrNotFound
	}
	return e, err
}

func (r *TrailRepository) List(ctx context.Context) ([]trail.Trail, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+trailColumns+" FROM trail ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []trail.Trail
	for rows.Next() {
		e, err := scanTrail(rows)
		if err != nil {
			return nil, err
		}
		entities = append(entities, e)
	}
	return entities, rows.Err()
}

func (r *TrailRepository) Create(ctx context.Context, e trail.Trail) (trail.Trail, error) {
	row := r.pool.QueryRow(ctx,
		"INSERT INTO trail (user_id, name, description, difficulty, distance_m, elevation_gain_m, elevation_loss_m, estimated_duration_s, path, is_official, is_published) VALUES ($1, $2, $3, COALESCE(NULLIF($4, '')::trail_difficulty, 'moderate'), $5, $6, $7, $8, ST_GeomFromText($9, 4326), $10, $11) RETURNING "+trailColumns,
		e.UserID, e.Name, e.Description, e.Difficulty, e.DistanceM, e.ElevationGainM, e.ElevationLossM, e.EstimatedDurationS, wktArg(e.Path), e.IsOfficial, e.IsPublished)
	return scanTrail(row)
}

func (r *TrailRepository) Update(ctx context.Context, e trail.Trail) (trail.Trail, error) {
	row := r.pool.QueryRow(ctx,
		"UPDATE trail SET user_id = $2, name = $3, description = $4, difficulty = COALESCE(NULLIF($5, '')::trail_difficulty, difficulty), distance_m = $6, elevation_gain_m = $7, elevation_loss_m = $8, estimated_duration_s = $9, path = ST_GeomFromText($10, 4326), is_official = $11, is_published = $12, updated_at = now() WHERE id = $1 RETURNING "+trailColumns,
		e.ID, e.UserID, e.Name, e.Description, e.Difficulty, e.DistanceM, e.ElevationGainM, e.ElevationLossM, e.EstimatedDurationS, wktArg(e.Path), e.IsOfficial, e.IsPublished)
	updated, err := scanTrail(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return trail.Trail{}, trail.ErrNotFound
	}
	return updated, err
}

func (r *TrailRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM trail WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return trail.ErrNotFound
	}
	return nil
}

func scanTrail(row pgx.Row) (trail.Trail, error) {
	var e trail.Trail
	var geomPath *string
	err := row.Scan(&e.ID, &e.UserID, &e.Name, &e.Description, &e.Difficulty, &e.DistanceM, &e.ElevationGainM, &e.ElevationLossM, &e.EstimatedDurationS, &geomPath, &e.IsOfficial, &e.IsPublished, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return trail.Trail{}, err
	}
	e.Path = wktBytes(geomPath)
	return e, nil
}
