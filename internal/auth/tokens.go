package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/traily-org/server/internal/domain/user"
)

// refreshTokenBytes is the amount of entropy of an opaque refresh token.
const refreshTokenBytes = 32

// jtiBytes is the amount of entropy of the "jti" claim of an access token.
const jtiBytes = 16

// accessClaims is the JWT payload of an access token.
type accessClaims struct {
	Role user.Role `json:"role"`
	jwt.RegisteredClaims
}

// signAccessToken returns an HS256 access token for the given user, valid
// from now until now+AccessTTL, and its expiry.
func signAccessToken(cfg Config, userID string, role user.Role, now time.Time) (string, time.Time, error) {
	jti, err := randomString(jtiBytes)
	if err != nil {
		return "", time.Time{}, err
	}

	expiresAt := now.Add(cfg.AccessTTL)
	claims := accessClaims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    cfg.Issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        jti,
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(cfg.AccessSecret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}
	return signed, expiresAt, nil
}

// parseAccessToken verifies an access token (HS256 signature only, issuer,
// expiry and subject) and returns its claims. Every failure is reported as
// ErrInvalidToken, without detail, so that callers cannot leak why a token was
// refused.
func parseAccessToken(cfg Config, token string, now func() time.Time) (Claims, error) {
	if token == "" {
		return Claims{}, ErrInvalidToken
	}

	var claims accessClaims
	parsed, err := jwt.ParseWithClaims(token, &claims,
		func(*jwt.Token) (any, error) { return cfg.AccessSecret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(cfg.Issuer),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithTimeFunc(now),
	)
	if err != nil || !parsed.Valid {
		return Claims{}, ErrInvalidToken
	}
	if claims.Subject == "" || !validRole(claims.Role) {
		return Claims{}, ErrInvalidToken
	}

	return Claims{UserID: claims.Subject, Role: claims.Role}, nil
}

// validRole reports whether r is one of the roles allowed in users.role.
func validRole(r user.Role) bool {
	switch r {
	case user.RoleUser, user.RoleModerator, user.RoleAdmin:
		return true
	default:
		return false
	}
}

// newRefreshToken returns a new opaque refresh token: 32 random bytes encoded
// as unpadded base64url.
func newRefreshToken() (string, error) {
	return randomString(refreshTokenBytes)
}

// hashRefreshToken returns the hex-encoded HMAC-SHA256 of token under secret.
// Only this hash is stored, so a database leak does not expose usable tokens.
func hashRefreshToken(secret []byte, token string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(token))
	return hex.EncodeToString(mac.Sum(nil))
}

// randomString returns n bytes from crypto/rand encoded as unpadded base64url.
func randomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
