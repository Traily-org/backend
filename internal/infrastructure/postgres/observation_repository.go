package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/observation"
)

const observationColumns = "id, user_id, species_id, photo_id, trail_id, track_id, ST_AsText(location), altitude, confidence_score, validation_status, visibility, observed_at, created_at, updated_at"

type ObservationRepository struct {
	pool *pgxpool.Pool
}

func NewObservationRepository(pool *pgxpool.Pool) *ObservationRepository {
	return &ObservationRepository{pool: pool}
}

func (r *ObservationRepository) GetByID(ctx context.Context, id string) (observation.Observation, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+observationColumns+" FROM observation WHERE id = $1", id)
	e, err := scanObservation(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return observation.Observation{}, observation.ErrNotFound
	}
	return e, err
}

func (r *ObservationRepository) List(ctx context.Context) ([]observation.Observation, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+observationColumns+" FROM observation ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []observation.Observation
	for rows.Next() {
		e, err := scanObservation(rows)
		if err != nil {
			return nil, err
		}
		entities = append(entities, e)
	}
	return entities, rows.Err()
}

func (r *ObservationRepository) Create(ctx context.Context, e observation.Observation) (observation.Observation, error) {
	row := r.pool.QueryRow(ctx,
		"INSERT INTO observation (user_id, species_id, photo_id, trail_id, track_id, location, altitude, confidence_score, validation_status, visibility, observed_at) VALUES ($1, $2, $3, $4, $5, ST_GeomFromText($6, 4326), $7, $8, COALESCE(NULLIF($9, '')::validation_status, 'pending'), COALESCE(NULLIF($10, '')::visibility_level, 'public'), COALESCE($11, now())) RETURNING "+observationColumns,
		e.UserID, e.SpeciesID, e.PhotoID, e.TrailID, e.TrackID, wktArg(e.Location), e.Altitude, e.ConfidenceScore, e.ValidationStatus, e.Visibility, e.ObservedAt)
	return scanObservation(row)
}

func (r *ObservationRepository) Update(ctx context.Context, e observation.Observation) (observation.Observation, error) {
	row := r.pool.QueryRow(ctx,
		"UPDATE observation SET user_id = $2, species_id = $3, photo_id = $4, trail_id = $5, track_id = $6, location = ST_GeomFromText($7, 4326), altitude = $8, confidence_score = $9, validation_status = COALESCE(NULLIF($10, '')::validation_status, validation_status), visibility = COALESCE(NULLIF($11, '')::visibility_level, visibility), observed_at = COALESCE($12, observed_at), updated_at = now() WHERE id = $1 RETURNING "+observationColumns,
		e.ID, e.UserID, e.SpeciesID, e.PhotoID, e.TrailID, e.TrackID, wktArg(e.Location), e.Altitude, e.ConfidenceScore, e.ValidationStatus, e.Visibility, e.ObservedAt)
	updated, err := scanObservation(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return observation.Observation{}, observation.ErrNotFound
	}
	return updated, err
}

func (r *ObservationRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM observation WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return observation.ErrNotFound
	}
	return nil
}

func scanObservation(row pgx.Row) (observation.Observation, error) {
	var e observation.Observation
	var geomLocation *string
	err := row.Scan(&e.ID, &e.UserID, &e.SpeciesID, &e.PhotoID, &e.TrailID, &e.TrackID, &geomLocation, &e.Altitude, &e.ConfidenceScore, &e.ValidationStatus, &e.Visibility, &e.ObservedAt, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return observation.Observation{}, err
	}
	e.Location = wktBytes(geomLocation)
	return e, nil
}
