// Package auth implements authentication: registration, login, short-lived
// JWT access tokens and revocable refresh tokens stored as hashes.
package auth

import (
	"context"
	"errors"
	"time"

	"github.com/traily-org/server/internal/domain/user"
	"github.com/traily-org/server/internal/pkg/apperror"
)

var (
	// ErrInvalidToken is returned when an access token is missing, malformed,
	// badly signed or expired.
	ErrInvalidToken = apperror.Unauthorized("invalid or expired access token")
	// ErrInvalidRefreshToken is returned when a refresh token is unknown,
	// revoked or expired.
	ErrInvalidRefreshToken = apperror.Unauthorized("invalid or expired refresh token")
	// ErrRefreshTokenNotFound is returned by a RefreshTokenRepository when no
	// token matches the given hash.
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
)

// Config holds the secrets and lifetimes used to issue tokens.
type Config struct {
	// AccessSecret signs access tokens (HS256).
	AccessSecret []byte
	// RefreshSecret is the HMAC key used to hash refresh tokens before storage.
	RefreshSecret []byte
	// AccessTTL is the lifetime of an access token (15 minutes in production).
	AccessTTL time.Duration
	// RefreshTTL is the lifetime of a refresh token (30 days in production).
	RefreshTTL time.Duration
	// Issuer is written to and checked in the "iss" claim.
	Issuer string
}

// Claims is what an access token proves about its bearer.
type Claims struct {
	UserID string
	Role   user.Role
}

// TokenPair is returned on register, login and refresh.
type TokenPair struct {
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
}

// RefreshToken is a stored refresh token. Only its hash is persisted.
type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

// UserStore is the subset of the user domain service auth relies on.
type UserStore interface {
	Create(ctx context.Context, u user.User) (user.User, error)
	Authenticate(ctx context.Context, email, password string) (user.User, error)
	Get(ctx context.Context, id string) (user.User, error)
}

// RefreshTokenRepository persists refresh tokens.
type RefreshTokenRepository interface {
	// Create stores t and returns it with its generated ID and CreatedAt.
	Create(ctx context.Context, t RefreshToken) (RefreshToken, error)
	// GetByHash returns the token with the given hash, or
	// ErrRefreshTokenNotFound.
	GetByHash(ctx context.Context, hash string) (RefreshToken, error)
	// Revoke marks the token as revoked at the given time. It reports whether
	// the token was still active, so that two concurrent refreshes cannot both
	// succeed.
	Revoke(ctx context.Context, id string, at time.Time) (bool, error)
	// RevokeAllForUser revokes every active token of the user.
	RevokeAllForUser(ctx context.Context, userID string, at time.Time) error
}
