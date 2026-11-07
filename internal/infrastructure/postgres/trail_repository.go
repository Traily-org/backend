package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/trail"
	"github.com/traily-org/server/internal/pkg/apperror"
)

// TrailRepository is the PostgreSQL implementation of trail.Repository.
type TrailRepository struct {
	pool *pgxpool.Pool
}

// NewTrailRepository returns a TrailRepository backed by the given pool.
func NewTrailRepository(pool *pgxpool.Pool) *TrailRepository {
	return &TrailRepository{pool: pool}
}

// GetByID returns the trail identified by id, or trail.ErrNotFound.
func (r *TrailRepository) GetByID(ctx context.Context, id string) (trail.Trail, error) {
	t, err := scanTrail(r.pool.QueryRow(ctx, getTrailQuery, id))
	if err != nil {
		return trail.Trail{}, mapError(err, trail.ErrNotFound)
	}

	return t, nil
}

// ListByUserID returns the trails owned by userID, most recent first.
func (r *TrailRepository) ListByUserID(ctx context.Context, userID string) ([]trail.Trail, error) {
	rows, err := r.pool.Query(ctx, listTrailsByUserQuery, userID)
	if err != nil {
		return nil, mapError(err, trail.ErrNotFound)
	}
	defer rows.Close()

	trails := []trail.Trail{}
	for rows.Next() {
		t, err := scanTrail(rows)
		if err != nil {
			return nil, mapError(err, trail.ErrNotFound)
		}
		trails = append(trails, t)
	}
	if err := rows.Err(); err != nil {
		return nil, mapError(err, trail.ErrNotFound)
	}

	return trails, nil
}

// Create inserts t and returns the stored trail. A user_id that does not
// reference an existing user is reported as a validation error.
func (r *TrailRepository) Create(ctx context.Context, t trail.Trail) (trail.Trail, error) {
	row := r.pool.QueryRow(ctx, createTrailQuery,
		t.UserID,
		t.Name,
		t.Description,
		string(t.Difficulty),
		t.DistanceMeters,
		t.ElevationGainMeters,
		t.ElevationLossMeters,
		t.EstimatedDurationSeconds,
		lineStringWKT(t.Path),
		t.IsPublished,
	)

	created, err := scanTrail(row)
	if err != nil {
		return trail.Trail{}, mapError(err, trail.ErrNotFound)
	}

	return created, nil
}

func scanTrail(row pgx.Row) (trail.Trail, error) {
	var (
		t          trail.Trail
		difficulty string
		geoJSON    string
	)
	err := row.Scan(
		&t.ID,
		&t.UserID,
		&t.Name,
		&t.Description,
		&difficulty,
		&t.DistanceMeters,
		&t.ElevationGainMeters,
		&t.ElevationLossMeters,
		&t.EstimatedDurationSeconds,
		&geoJSON,
		&t.IsOfficial,
		&t.IsPublished,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err != nil {
		return trail.Trail{}, err
	}

	t.Difficulty = trail.Difficulty(difficulty)
	t.Path, err = pathFromGeoJSON(geoJSON)
	if err != nil {
		return trail.Trail{}, apperror.Internal(err)
	}

	return t, nil
}

// lineStringWKT renders path as a WKT LINESTRING ("lon lat" pairs).
func lineStringWKT(path []trail.Coordinate) string {
	points := make([]string, len(path))
	for i, c := range path {
		points[i] = strconv.FormatFloat(c.Longitude, 'f', -1, 64) + " " + strconv.FormatFloat(c.Latitude, 'f', -1, 64)
	}

	return "LINESTRING(" + strings.Join(points, ", ") + ")"
}

// pathFromGeoJSON parses the GeoJSON LineString produced by ST_AsGeoJSON.
func pathFromGeoJSON(geoJSON string) ([]trail.Coordinate, error) {
	var line struct {
		Type        string      `json:"type"`
		Coordinates [][]float64 `json:"coordinates"`
	}
	if err := json.Unmarshal([]byte(geoJSON), &line); err != nil {
		return nil, fmt.Errorf("parse trail path: %w", err)
	}
	if line.Type != "LineString" {
		return nil, fmt.Errorf("parse trail path: unexpected geometry type %q", line.Type)
	}

	path := make([]trail.Coordinate, len(line.Coordinates))
	for i, c := range line.Coordinates {
		if len(c) < 2 {
			return nil, fmt.Errorf("parse trail path: point %d has %d values", i, len(c))
		}
		path[i] = trail.Coordinate{Longitude: c[0], Latitude: c[1]}
	}

	return path, nil
}
