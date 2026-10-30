package auth

import (
	"context"
	"errors"
	"time"

	"github.com/traily-org/server/internal/domain/user"
	"github.com/traily-org/server/internal/pkg/apperror"
)

// Service registers and logs users in, and issues, rotates and revokes their
// tokens. Access tokens are stateless JWTs; refresh tokens are opaque, stored
// as HMAC hashes and rotated on every use.
type Service struct {
	users  UserStore
	tokens RefreshTokenRepository
	cfg    Config
	now    func() time.Time
}

// NewService returns a Service issuing tokens with cfg.
func NewService(users UserStore, tokens RefreshTokenRepository, cfg Config) *Service {
	return &Service{
		users:  users,
		tokens: tokens,
		cfg:    cfg,
		now:    time.Now,
	}
}

// Register creates u through the user store and returns the created user with
// a new token pair. Errors from the user store are returned unchanged.
func (s *Service) Register(ctx context.Context, u user.User) (user.User, TokenPair, error) {
	created, err := s.users.Create(ctx, u)
	if err != nil {
		return user.User{}, TokenPair{}, err
	}

	pair, err := s.issuePair(ctx, created)
	if err != nil {
		return user.User{}, TokenPair{}, err
	}
	return created, pair, nil
}

// Login checks the credentials through the user store and returns the user
// with a new token pair. Errors from the user store are returned unchanged.
func (s *Service) Login(ctx context.Context, email, password string) (user.User, TokenPair, error) {
	u, err := s.users.Authenticate(ctx, email, password)
	if err != nil {
		return user.User{}, TokenPair{}, err
	}

	pair, err := s.issuePair(ctx, u)
	if err != nil {
		return user.User{}, TokenPair{}, err
	}
	return u, pair, nil
}

// Refresh exchanges a valid refresh token for a new token pair and revokes the
// presented one (rotation). It returns ErrInvalidRefreshToken when the token is
// unknown, expired or belongs to a deleted user. Presenting a token that was
// already revoked is treated as a theft: every token of its owner is revoked
// and ErrInvalidRefreshToken is returned.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (TokenPair, error) {
	if refreshToken == "" {
		return TokenPair{}, ErrInvalidRefreshToken
	}

	stored, err := s.tokens.GetByHash(ctx, hashRefreshToken(s.cfg.RefreshSecret, refreshToken))
	if errors.Is(err, ErrRefreshTokenNotFound) {
		return TokenPair{}, ErrInvalidRefreshToken
	}
	if err != nil {
		return TokenPair{}, err
	}

	now := s.now()
	if stored.RevokedAt != nil {
		return TokenPair{}, s.revokeAllOnReuse(ctx, stored.UserID, now)
	}
	if !now.Before(stored.ExpiresAt) {
		return TokenPair{}, ErrInvalidRefreshToken
	}

	revoked, err := s.tokens.Revoke(ctx, stored.ID, now)
	if err != nil {
		return TokenPair{}, err
	}
	if !revoked {
		// Another request revoked it between GetByHash and Revoke: the token
		// was used twice.
		return TokenPair{}, s.revokeAllOnReuse(ctx, stored.UserID, now)
	}

	u, err := s.users.Get(ctx, stored.UserID)
	if errors.Is(err, user.ErrNotFound) {
		return TokenPair{}, ErrInvalidRefreshToken
	}
	if err != nil {
		return TokenPair{}, err
	}

	return s.issuePair(ctx, u)
}

// Logout revokes the given refresh token. It is idempotent: an empty, unknown
// or already revoked token is not an error.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}

	stored, err := s.tokens.GetByHash(ctx, hashRefreshToken(s.cfg.RefreshSecret, refreshToken))
	if errors.Is(err, ErrRefreshTokenNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if stored.RevokedAt != nil {
		return nil
	}

	if _, err := s.tokens.Revoke(ctx, stored.ID, s.now()); err != nil {
		return err
	}
	return nil
}

// ParseAccessToken verifies an access token and returns its claims. Only
// HS256 signatures made with the access secret are accepted, and the token
// must carry the configured issuer, a subject, a known role and an expiry in
// the future. Every failure returns ErrInvalidToken.
func (s *Service) ParseAccessToken(token string) (Claims, error) {
	return parseAccessToken(s.cfg, token, s.now)
}

// issuePair signs an access token for u and stores a new refresh token.
func (s *Service) issuePair(ctx context.Context, u user.User) (TokenPair, error) {
	now := s.now()

	access, accessExp, err := signAccessToken(s.cfg, u.ID, u.Role, now)
	if err != nil {
		return TokenPair{}, apperror.Internal(err)
	}

	refresh, err := newRefreshToken()
	if err != nil {
		return TokenPair{}, apperror.Internal(err)
	}

	refreshExp := now.Add(s.cfg.RefreshTTL)
	if _, err := s.tokens.Create(ctx, RefreshToken{
		UserID:    u.ID,
		TokenHash: hashRefreshToken(s.cfg.RefreshSecret, refresh),
		ExpiresAt: refreshExp,
	}); err != nil {
		return TokenPair{}, err
	}

	return TokenPair{
		AccessToken:      access,
		AccessExpiresAt:  accessExp,
		RefreshToken:     refresh,
		RefreshExpiresAt: refreshExp,
	}, nil
}

// revokeAllOnReuse revokes every token of userID after a refresh token was
// replayed, and returns the error to give back to the caller.
func (s *Service) revokeAllOnReuse(ctx context.Context, userID string, now time.Time) error {
	if err := s.tokens.RevokeAllForUser(ctx, userID, now); err != nil {
		return err
	}
	return ErrInvalidRefreshToken
}

// RevokeAllSessions revokes every refresh token of the user. It implements
// user.SessionRevoker.
func (s *Service) RevokeAllSessions(ctx context.Context, userID string) error {
	return s.tokens.RevokeAllForUser(ctx, userID, s.now())
}
