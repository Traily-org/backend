package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/auth"
)

const (
	refreshTokenColumns = `id::text, user_id::text, token_hash, expires_at, revoked_at, created_at`

	createRefreshTokenQuery = `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING ` + refreshTokenColumns

	getRefreshTokenByHashQuery = `
		SELECT ` + refreshTokenColumns + `
		FROM refresh_tokens
		WHERE token_hash = $1`

	revokeRefreshTokenQuery = `
		UPDATE refresh_tokens
		SET revoked_at = $2
		WHERE id = $1 AND revoked_at IS NULL`

	revokeUserRefreshTokensQuery = `
		UPDATE refresh_tokens
		SET revoked_at = $2
		WHERE user_id = $1 AND revoked_at IS NULL`
)

// RefreshTokenRepository is the PostgreSQL implementation of
// auth.RefreshTokenRepository. It only ever sees token hashes.
type RefreshTokenRepository struct {
	pool *pgxpool.Pool
}

// NewRefreshTokenRepository returns a RefreshTokenRepository backed by the
// given pool.
func NewRefreshTokenRepository(pool *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{pool: pool}
}

// Create stores t and returns it with its generated ID and CreatedAt.
func (r *RefreshTokenRepository) Create(ctx context.Context, t auth.RefreshToken) (auth.RefreshToken, error) {
	row := r.pool.QueryRow(ctx, createRefreshTokenQuery, t.UserID, t.TokenHash, t.ExpiresAt)

	created, err := scanRefreshToken(row)
	if err != nil {
		return auth.RefreshToken{}, mapError(err, auth.ErrRefreshTokenNotFound)
	}
	return created, nil
}

// GetByHash returns the token with the given hash, or
// auth.ErrRefreshTokenNotFound.
func (r *RefreshTokenRepository) GetByHash(ctx context.Context, hash string) (auth.RefreshToken, error) {
	row := r.pool.QueryRow(ctx, getRefreshTokenByHashQuery, hash)

	t, err := scanRefreshToken(row)
	if err != nil {
		return auth.RefreshToken{}, mapError(err, auth.ErrRefreshTokenNotFound)
	}
	return t, nil
}

// Revoke marks the token as revoked at the given time, atomically. It reports
// whether the token was still active; false means it was already revoked or
// does not exist.
func (r *RefreshTokenRepository) Revoke(ctx context.Context, id string, at time.Time) (bool, error) {
	tag, err := r.pool.Exec(ctx, revokeRefreshTokenQuery, id, at)
	if err != nil {
		return false, mapError(err, auth.ErrRefreshTokenNotFound)
	}
	return tag.RowsAffected() == 1, nil
}

// RevokeAllForUser revokes every active token of the user. Tokens already
// revoked keep their original revocation time.
func (r *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID string, at time.Time) error {
	if _, err := r.pool.Exec(ctx, revokeUserRefreshTokensQuery, userID, at); err != nil {
		return mapError(err, auth.ErrRefreshTokenNotFound)
	}
	return nil
}

// scanRefreshToken reads a row selected with refreshTokenColumns.
func scanRefreshToken(row pgx.Row) (auth.RefreshToken, error) {
	var t auth.RefreshToken
	err := row.Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.RevokedAt, &t.CreatedAt)
	return t, err
}
