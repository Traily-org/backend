package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/refreshtoken"
)

const refreshtokenColumns = "id, user_id, token_hash, expires_at, revoked_at, created_at"

type RefreshTokenRepository struct {
	pool *pgxpool.Pool
}

func NewRefreshTokenRepository(pool *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{pool: pool}
}

func (r *RefreshTokenRepository) GetByID(ctx context.Context, id string) (refreshtoken.RefreshToken, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+refreshtokenColumns+" FROM refresh_token WHERE id = $1", id)
	e, err := scanRefreshToken(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return refreshtoken.RefreshToken{}, refreshtoken.ErrNotFound
	}
	return e, err
}

func (r *RefreshTokenRepository) List(ctx context.Context) ([]refreshtoken.RefreshToken, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+refreshtokenColumns+" FROM refresh_token ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []refreshtoken.RefreshToken
	for rows.Next() {
		e, err := scanRefreshToken(rows)
		if err != nil {
			return nil, err
		}
		entities = append(entities, e)
	}
	return entities, rows.Err()
}

func (r *RefreshTokenRepository) Create(ctx context.Context, e refreshtoken.RefreshToken) (refreshtoken.RefreshToken, error) {
	row := r.pool.QueryRow(ctx,
		"INSERT INTO refresh_token (user_id, token_hash, expires_at, revoked_at) VALUES ($1, $2, $3, $4) RETURNING "+refreshtokenColumns,
		e.UserID, e.TokenHash, e.ExpiresAt, e.RevokedAt)
	return scanRefreshToken(row)
}

func (r *RefreshTokenRepository) Update(ctx context.Context, e refreshtoken.RefreshToken) (refreshtoken.RefreshToken, error) {
	row := r.pool.QueryRow(ctx,
		"UPDATE refresh_token SET user_id = $2, token_hash = $3, expires_at = $4, revoked_at = $5 WHERE id = $1 RETURNING "+refreshtokenColumns,
		e.ID, e.UserID, e.TokenHash, e.ExpiresAt, e.RevokedAt)
	updated, err := scanRefreshToken(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return refreshtoken.RefreshToken{}, refreshtoken.ErrNotFound
	}
	return updated, err
}

func (r *RefreshTokenRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM refresh_token WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return refreshtoken.ErrNotFound
	}
	return nil
}

func scanRefreshToken(row pgx.Row) (refreshtoken.RefreshToken, error) {
	var e refreshtoken.RefreshToken
	err := row.Scan(&e.ID, &e.UserID, &e.TokenHash, &e.ExpiresAt, &e.RevokedAt, &e.CreatedAt)
	if err != nil {
		return refreshtoken.RefreshToken{}, err
	}
	return e, nil
}
