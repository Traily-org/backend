package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/moderationlog"
)

const moderationlogColumns = "id, moderator_id, action, target_type, target_id, notes, created_at"

type ModerationLogRepository struct {
	pool *pgxpool.Pool
}

func NewModerationLogRepository(pool *pgxpool.Pool) *ModerationLogRepository {
	return &ModerationLogRepository{pool: pool}
}

func (r *ModerationLogRepository) GetByID(ctx context.Context, id string) (moderationlog.ModerationLog, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+moderationlogColumns+" FROM moderation_log WHERE id = $1", id)
	e, err := scanModerationLog(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return moderationlog.ModerationLog{}, moderationlog.ErrNotFound
	}
	return e, err
}

func (r *ModerationLogRepository) List(ctx context.Context) ([]moderationlog.ModerationLog, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+moderationlogColumns+" FROM moderation_log ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []moderationlog.ModerationLog
	for rows.Next() {
		e, err := scanModerationLog(rows)
		if err != nil {
			return nil, err
		}
		entities = append(entities, e)
	}
	return entities, rows.Err()
}

func (r *ModerationLogRepository) Create(ctx context.Context, e moderationlog.ModerationLog) (moderationlog.ModerationLog, error) {
	row := r.pool.QueryRow(ctx,
		"INSERT INTO moderation_log (moderator_id, action, target_type, target_id, notes) VALUES ($1, $2, $3, $4, $5) RETURNING "+moderationlogColumns,
		e.ModeratorID, e.Action, e.TargetType, e.TargetID, e.Notes)
	return scanModerationLog(row)
}

func (r *ModerationLogRepository) Update(ctx context.Context, e moderationlog.ModerationLog) (moderationlog.ModerationLog, error) {
	row := r.pool.QueryRow(ctx,
		"UPDATE moderation_log SET moderator_id = $2, action = COALESCE(NULLIF($3, '')::mod_action, action), target_type = COALESCE(NULLIF($4, '')::report_target, target_type), target_id = $5, notes = $6 WHERE id = $1 RETURNING "+moderationlogColumns,
		e.ID, e.ModeratorID, e.Action, e.TargetType, e.TargetID, e.Notes)
	updated, err := scanModerationLog(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return moderationlog.ModerationLog{}, moderationlog.ErrNotFound
	}
	return updated, err
}

func (r *ModerationLogRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM moderation_log WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return moderationlog.ErrNotFound
	}
	return nil
}

func scanModerationLog(row pgx.Row) (moderationlog.ModerationLog, error) {
	var e moderationlog.ModerationLog
	err := row.Scan(&e.ID, &e.ModeratorID, &e.Action, &e.TargetType, &e.TargetID, &e.Notes, &e.CreatedAt)
	if err != nil {
		return moderationlog.ModerationLog{}, err
	}
	return e, nil
}
