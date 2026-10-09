package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/activity"
)

type ActivityRepository struct{ pool *pgxpool.Pool }

func NewActivityRepository(pool *pgxpool.Pool) *ActivityRepository {
	return &ActivityRepository{pool: pool}
}
func (r *ActivityRepository) GetByID(ctx context.Context, id string) (activity.Activity, error) {
	row := r.pool.QueryRow(ctx, "SELECT id, user_id, track_id, trace_id, name, started_at, ended_at, created_at FROM activity WHERE id = $1", id)
	var a activity.Activity
	err := row.Scan(&a.ID, &a.UserID, &a.TrackID, &a.TraceID, &a.Name, &a.StartedAt, &a.EndedAt, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return activity.Activity{}, activity.ErrNotFound
	}
	return a, err
}
func (r *ActivityRepository) List(ctx context.Context) ([]activity.Activity, error) {
	rows, err := r.pool.Query(ctx, "SELECT id, user_id, track_id, trace_id, name, started_at, ended_at, created_at FROM activity ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var activities []activity.Activity
	for rows.Next() {
		var a activity.Activity
		if err := rows.Scan(&a.ID, &a.UserID, &a.TrackID, &a.TraceID, &a.Name, &a.StartedAt, &a.EndedAt, &a.CreatedAt); err != nil {
			return nil, err
		}
		activities = append(activities, a)
	}
	return activities, rows.Err()
}
func (r *ActivityRepository) Create(ctx context.Context, a activity.Activity) (activity.Activity, error) {
	row := r.pool.QueryRow(ctx, "INSERT INTO activity (user_id, track_id, trace_id, name, started_at, ended_at) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, user_id, track_id, trace_id, name, started_at, ended_at, created_at", a.UserID, a.TrackID, a.TraceID, a.Name, a.StartedAt, a.EndedAt)
	err := row.Scan(&a.ID, &a.UserID, &a.TrackID, &a.TraceID, &a.Name, &a.StartedAt, &a.EndedAt, &a.CreatedAt)
	return a, err
}
func (r *ActivityRepository) Update(ctx context.Context, a activity.Activity) (activity.Activity, error) {
	row := r.pool.QueryRow(ctx, "UPDATE activity SET user_id=$2, track_id=$3, trace_id=$4, name=$5, started_at=$6, ended_at=$7 WHERE id=$1 RETURNING id, user_id, track_id, trace_id, name, started_at, ended_at, created_at", a.ID, a.UserID, a.TrackID, a.TraceID, a.Name, a.StartedAt, a.EndedAt)
	err := row.Scan(&a.ID, &a.UserID, &a.TrackID, &a.TraceID, &a.Name, &a.StartedAt, &a.EndedAt, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return activity.Activity{}, activity.ErrNotFound
	}
	return a, err
}
func (r *ActivityRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM activity WHERE id=$1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return activity.ErrNotFound
	}
	return nil
}
