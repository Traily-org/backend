package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/observation"
)

const observationColumns = "id, user_id, species_id, photo_id, trace_id, activity_id, ST_AsText(location), altitude, confidence_score, validation_status, visibility, observed_at, created_at, updated_at"

type ObservationRepository struct{ pool *pgxpool.Pool }

func NewObservationRepository(pool *pgxpool.Pool) *ObservationRepository {
	return &ObservationRepository{pool: pool}
}

func scanObservation(row pgx.Row) (observation.Observation, error) {
	var o observation.Observation
	var loc string
	err := row.Scan(&o.ID, &o.UserID, &o.SpeciesID, &o.PhotoID, &o.TraceID, &o.ActivityID, &loc, &o.Altitude, &o.ConfidenceScore, &o.ValidationStatus, &o.Visibility, &o.ObservedAt, &o.CreatedAt, &o.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return observation.Observation{}, observation.ErrNotFound
	}
	o.Location = []byte(loc)
	return o, err
}

func (r *ObservationRepository) GetByID(ctx context.Context, id string) (observation.Observation, error) {
	return scanObservation(r.pool.QueryRow(ctx, "SELECT "+observationColumns+" FROM observations WHERE id = $1", id))
}

func (r *ObservationRepository) List(ctx context.Context) ([]observation.Observation, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+observationColumns+" FROM observations ORDER BY observed_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []observation.Observation
	for rows.Next() {
		o, err := scanObservation(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, o)
	}
	return list, rows.Err()
}

func (r *ObservationRepository) Create(ctx context.Context, o observation.Observation) (observation.Observation, error) {
	return scanObservation(r.pool.QueryRow(ctx, "INSERT INTO observations (user_id, species_id, photo_id, trace_id, activity_id, location, altitude, confidence_score, validation_status, visibility, observed_at) VALUES ($1, $2, $3, $4, $5, ST_GeomFromText($6, 4326), $7, $8, $9, $10, $11) RETURNING "+observationColumns,
		o.UserID, o.SpeciesID, o.PhotoID, o.TraceID, o.ActivityID, string(o.Location), o.Altitude, o.ConfidenceScore, o.ValidationStatus, o.Visibility, o.ObservedAt))
}

func (r *ObservationRepository) Update(ctx context.Context, o observation.Observation) (observation.Observation, error) {
	return scanObservation(r.pool.QueryRow(ctx, "UPDATE observations SET user_id=$2, species_id=$3, photo_id=$4, trace_id=$5, activity_id=$6, location=ST_GeomFromText($7, 4326), altitude=$8, confidence_score=$9, validation_status=$10, visibility=$11, observed_at=$12, updated_at=NOW() WHERE id=$1 RETURNING "+observationColumns,
		o.ID, o.UserID, o.SpeciesID, o.PhotoID, o.TraceID, o.ActivityID, string(o.Location), o.Altitude, o.ConfidenceScore, o.ValidationStatus, o.Visibility, o.ObservedAt))
}

func (r *ObservationRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM observations WHERE id=$1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return observation.ErrNotFound
	}
	return nil
}
