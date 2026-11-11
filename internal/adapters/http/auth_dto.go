package http

import (
	"time"

	"github.com/traily-org/server/internal/auth"
	"github.com/traily-org/server/internal/domain/user"
)

// tokenTypeBearer is the token_type of every issued access token.
const tokenTypeBearer = "Bearer"

// registerRequest is the payload of POST /auth/register.
type registerRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
	Name     string `json:"name" binding:"required"`
}

// loginRequest is the payload of POST /auth/login.
type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// refreshRequest is the payload of POST /auth/refresh.
type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// logoutRequest is the payload of POST /auth/logout.
type logoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// tokensResponse is the public representation of a token pair. Dates are UTC.
type tokensResponse struct {
	AccessToken      string    `json:"access_token"`
	AccessExpiresAt  time.Time `json:"access_expires_at"`
	RefreshToken     string    `json:"refresh_token"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
	TokenType        string    `json:"token_type"`
}

func newTokensResponse(p auth.TokenPair) tokensResponse {
	return tokensResponse{
		AccessToken:      p.AccessToken,
		AccessExpiresAt:  p.AccessExpiresAt.UTC(),
		RefreshToken:     p.RefreshToken,
		RefreshExpiresAt: p.RefreshExpiresAt.UTC(),
		TokenType:        tokenTypeBearer,
	}
}

// authResponse is returned by register and login.
type authResponse struct {
	User   userResponse   `json:"user"`
	Tokens tokensResponse `json:"tokens"`
}

func newAuthResponse(u user.User, p auth.TokenPair) authResponse {
	return authResponse{
		User:   newUserResponse(u),
		Tokens: newTokensResponse(p),
	}
}

// refreshResponse is returned by refresh.
type refreshResponse struct {
	Tokens tokensResponse `json:"tokens"`
}
