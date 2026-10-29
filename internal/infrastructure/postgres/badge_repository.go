package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/badge"
)

const badgeColumns = "id, code, name, description, icon_url, criteria, created_at"

type BadgeRepository struct {
	pool *pgxpool.Pool
}

func NewBadgeRepository(pool *pgxpool.Pool) *BadgeRepository {
	return &BadgeRepository{pool: pool}
}

func (r *BadgeRepository) GetByID(ctx context.Context, id string) (badge.Badge, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+badgeColumns+" FROM badge WHERE id = $1", id)
	e, err := scanBadge(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return badge.Badge{}, badge.ErrNotFound
	}
	return e, err
}

func (r *BadgeRepository) List(ctx context.Context) ([]badge.Badge, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+badgeColumns+" FROM badge ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []badge.Badge
	for rows.Next() {
		e, err := scanBadge(rows)
		if err != nil {
			return nil, err
		}
		entities = append(entities, e)
	}
	return entities, rows.Err()
}

func (r *BadgeRepository) Create(ctx context.Context, e badge.Badge) (badge.Badge, error) {
	row := r.pool.QueryRow(ctx,
		"INSERT INTO badge (code, name, description, icon_url, criteria) VALUES ($1, $2, $3, $4, COALESCE($5::jsonb, '{}'::jsonb)) RETURNING "+badgeColumns,
		e.Code, e.Name, e.Description, e.IconURL, e.Criteria)
	return scanBadge(row)
}

func (r *BadgeRepository) Update(ctx context.Context, e badge.Badge) (badge.Badge, error) {
	row := r.pool.QueryRow(ctx,
		"UPDATE badge SET code = $2, name = $3, description = $4, icon_url = $5, criteria = COALESCE($6::jsonb, '{}'::jsonb) WHERE id = $1 RETURNING "+badgeColumns,
		e.ID, e.Code, e.Name, e.Description, e.IconURL, e.Criteria)
	updated, err := scanBadge(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return badge.Badge{}, badge.ErrNotFound
	}
	return updated, err
}

func (r *BadgeRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM badge WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return badge.ErrNotFound
	}
	return nil
}

func scanBadge(row pgx.Row) (badge.Badge, error) {
	var e badge.Badge
	err := row.Scan(&e.ID, &e.Code, &e.Name, &e.Description, &e.IconURL, &e.Criteria, &e.CreatedAt)
	if err != nil {
		return badge.Badge{}, err
	}
	return e, nil
}
