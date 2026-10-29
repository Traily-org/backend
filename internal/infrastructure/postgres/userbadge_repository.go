package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/userbadge"
)

const userbadgeColumns = "id, user_id, badge_id, unlocked_at"

type UserBadgeRepository struct {
	pool *pgxpool.Pool
}

func NewUserBadgeRepository(pool *pgxpool.Pool) *UserBadgeRepository {
	return &UserBadgeRepository{pool: pool}
}

func (r *UserBadgeRepository) GetByID(ctx context.Context, id string) (userbadge.UserBadge, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+userbadgeColumns+" FROM user_badge WHERE id = $1", id)
	e, err := scanUserBadge(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return userbadge.UserBadge{}, userbadge.ErrNotFound
	}
	return e, err
}

func (r *UserBadgeRepository) List(ctx context.Context) ([]userbadge.UserBadge, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+userbadgeColumns+" FROM user_badge ORDER BY unlocked_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []userbadge.UserBadge
	for rows.Next() {
		e, err := scanUserBadge(rows)
		if err != nil {
			return nil, err
		}
		entities = append(entities, e)
	}
	return entities, rows.Err()
}

func (r *UserBadgeRepository) Create(ctx context.Context, e userbadge.UserBadge) (userbadge.UserBadge, error) {
	row := r.pool.QueryRow(ctx,
		"INSERT INTO user_badge (user_id, badge_id) VALUES ($1, $2) RETURNING "+userbadgeColumns,
		e.UserID, e.BadgeID)
	return scanUserBadge(row)
}

func (r *UserBadgeRepository) Update(ctx context.Context, e userbadge.UserBadge) (userbadge.UserBadge, error) {
	row := r.pool.QueryRow(ctx,
		"UPDATE user_badge SET user_id = $2, badge_id = $3 WHERE id = $1 RETURNING "+userbadgeColumns,
		e.ID, e.UserID, e.BadgeID)
	updated, err := scanUserBadge(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return userbadge.UserBadge{}, userbadge.ErrNotFound
	}
	return updated, err
}

func (r *UserBadgeRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM user_badge WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return userbadge.ErrNotFound
	}
	return nil
}

func scanUserBadge(row pgx.Row) (userbadge.UserBadge, error) {
	var e userbadge.UserBadge
	err := row.Scan(&e.ID, &e.UserID, &e.BadgeID, &e.UnlockedAt)
	if err != nil {
		return userbadge.UserBadge{}, err
	}
	return e, nil
}
