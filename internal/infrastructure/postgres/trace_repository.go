package postgres

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/traily-org/server/internal/domain/trace"
)

type TraceRepository struct{ pool *pgxpool.Pool }

func NewTraceRepository(pool *pgxpool.Pool) *TraceRepository { return &TraceRepository{pool: pool} }
func (r *TraceRepository) GetByID(ctx context.Context, id string) (trace.Trace, error) {
	row := r.pool.QueryRow(ctx, "SELECT id, user_id, track_id, name, description, difficulty, created_at, updated_at FROM trace WHERE id = $1", id)
	var t trace.Trace
	err := row.Scan(&t.ID, &t.UserID, &t.TrackID, &t.Name, &t.Description, &t.Difficulty, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return trace.Trace{}, trace.ErrNotFound
	}
	return t, err
}
func (r *TraceRepository) List(ctx context.Context) ([]trace.Trace, error) {
	rows, err := r.pool.Query(ctx, "SELECT id, user_id, track_id, name, description, difficulty, created_at, updated_at FROM trace ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var traces []trace.Trace
	for rows.Next() {
		var t trace.Trace
		if err := rows.Scan(&t.ID, &t.UserID, &t.TrackID, &t.Name, &t.Description, &t.Difficulty, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		traces = append(traces, t)
	}
	return traces, rows.Err()
}
func (r *TraceRepository) Create(ctx context.Context, t trace.Trace) (trace.Trace, error) {
	row := r.pool.QueryRow(ctx, "INSERT INTO trace (user_id, track_id, name, description, difficulty) VALUES ($1, $2, $3, $4, $5) RETURNING id, user_id, track_id, name, description, difficulty, created_at, updated_at", t.UserID, t.TrackID, t.Name, t.Description, t.Difficulty)
	err := row.Scan(&t.ID, &t.UserID, &t.TrackID, &t.Name, &t.Description, &t.Difficulty, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}
func (r *TraceRepository) Update(ctx context.Context, t trace.Trace) (trace.Trace, error) {
	row := r.pool.QueryRow(ctx, "UPDATE trace SET user_id=$2, track_id=$3, name=$4, description=$5, difficulty=$6, updated_at=NOW() WHERE id=$1 RETURNING id, user_id, track_id, name, description, difficulty, created_at, updated_at", t.ID, t.UserID, t.TrackID, t.Name, t.Description, t.Difficulty)
	err := row.Scan(&t.ID, &t.UserID, &t.TrackID, &t.Name, &t.Description, &t.Difficulty, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return trace.Trace{}, trace.ErrNotFound
	}
	return t, err
}
func (r *TraceRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM trace WHERE id=$1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return trace.ErrNotFound
	}
	return nil
}
