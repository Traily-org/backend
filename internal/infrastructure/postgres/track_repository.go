package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/track"
)

type TrackRepository struct {
	pool *pgxpool.Pool
}

func NewTrackRepository(pool *pgxpool.Pool) *TrackRepository {
	return &TrackRepository{pool: pool}
}

func (r *TrackRepository) GetByID(ctx context.Context, id string) (track.Track, error) {
	row := r.pool.QueryRow(ctx, getTrackQuery, id)
	t, err := scanTrack(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return track.Track{}, track.ErrNotFound
	}
	return t, err
}

func (r *TrackRepository) ListByUserID(ctx context.Context, userID string) ([]track.Track, error) {
	rows, err := r.pool.Query(ctx, listTracksByUserQuery, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []track.Track
	for rows.Next() {
		t, err := scanTrack(rows)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, t)
	}
	return tracks, rows.Err()
}

func (r *TrackRepository) Create(ctx context.Context, t track.Track) (track.Track, error) {
	var routeWKT string
	if t.Route != nil {
		routeWKT = string(t.Route)
	}
	row := r.pool.QueryRow(ctx, createTrackQuery, t.UserID, routeWKT, t.DistanceM, t.ElevationGainM, t.ElevationLossM, t.StartedAt, t.EndedAt)
	return scanTrack(row)
}

func (r *TrackRepository) Update(ctx context.Context, t track.Track) (track.Track, error) {
	var routeWKT string
	if t.Route != nil {
		routeWKT = string(t.Route)
	}
	row := r.pool.QueryRow(ctx, updateTrackQuery, t.ID, t.UserID, routeWKT, t.DistanceM, t.ElevationGainM, t.ElevationLossM, t.StartedAt, t.EndedAt)
	updated, err := scanTrack(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return track.Track{}, track.ErrNotFound
	}
	return updated, err
}

func (r *TrackRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, deleteTrackQuery, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return track.ErrNotFound
	}
	return nil
}

func scanTrack(row pgx.Row) (track.Track, error) {
	var t track.Track
	var routeWKT string
	err := row.Scan(&t.ID, &t.UserID, &routeWKT, &t.DistanceM, &t.ElevationGainM, &t.ElevationLossM, &t.StartedAt, &t.EndedAt, &t.CreatedAt)
	if err == nil && routeWKT != "" {
		t.Route = []byte(routeWKT)
	}
	return t, err
}
