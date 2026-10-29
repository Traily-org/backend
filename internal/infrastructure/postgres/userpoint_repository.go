package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/userpoint"
)

const userpointColumns = "id, user_id, points, reason, reference_id, created_at"

type UserPointRepository struct {
	pool *pgxpool.Pool
}

func NewUserPointRepository(pool *pgxpool.Pool) *UserPointRepository {
	return &UserPointRepository{pool: pool}
}

func (r *UserPointRepository) GetByID(ctx context.Context, id string) (userpoint.UserPoint, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+userpointColumns+" FROM user_point WHERE id = $1", id)
	e, err := scanUserPoint(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return userpoint.UserPoint{}, userpoint.ErrNotFound
	}
	return e, err
}

func (r *UserPointRepository) List(ctx context.Context) ([]userpoint.UserPoint, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+userpointColumns+" FROM user_point ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []userpoint.UserPoint
	for rows.Next() {
		e, err := scanUserPoint(rows)
		if err != nil {
			return nil, err
		}
		entities = append(entities, e)
	}
	return entities, rows.Err()
}

func (r *UserPointRepository) Create(ctx context.Context, e userpoint.UserPoint) (userpoint.UserPoint, error) {
	row := r.pool.QueryRow(ctx,
		"INSERT INTO user_point (user_id, points, reason, reference_id) VALUES ($1, $2, $3, $4) RETURNING "+userpointColumns,
		e.UserID, e.Points, e.Reason, e.ReferenceID)
	return scanUserPoint(row)
}

func (r *UserPointRepository) Update(ctx context.Context, e userpoint.UserPoint) (userpoint.UserPoint, error) {
	row := r.pool.QueryRow(ctx,
		"UPDATE user_point SET user_id = $2, points = $3, reason = $4, reference_id = $5 WHERE id = $1 RETURNING "+userpointColumns,
		e.ID, e.UserID, e.Points, e.Reason, e.ReferenceID)
	updated, err := scanUserPoint(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return userpoint.UserPoint{}, userpoint.ErrNotFound
	}
	return updated, err
}

func (r *UserPointRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM user_point WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return userpoint.ErrNotFound
	}
	return nil
}

func scanUserPoint(row pgx.Row) (userpoint.UserPoint, error) {
	var e userpoint.UserPoint
	err := row.Scan(&e.ID, &e.UserID, &e.Points, &e.Reason, &e.ReferenceID, &e.CreatedAt)
	if err != nil {
		return userpoint.UserPoint{}, err
	}
	return e, nil
}
