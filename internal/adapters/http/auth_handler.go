package http

import (
	"context"
	nethttp "net/http"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/adapters/http/response"
	"github.com/traily-org/server/internal/auth"
	"github.com/traily-org/server/internal/domain/user"
)

// authService is the subset of the auth service used by AuthHandler.
type authService interface {
	Register(ctx context.Context, u user.User) (user.User, auth.TokenPair, error)
	Login(ctx context.Context, email, password string) (user.User, auth.TokenPair, error)
	Refresh(ctx context.Context, refreshToken string) (auth.TokenPair, error)
	Logout(ctx context.Context, refreshToken string) error
}

// AuthHandler serves the /auth endpoints.
type AuthHandler struct {
	service authService
}

// NewAuthHandler returns an AuthHandler backed by service.
func NewAuthHandler(service authService) *AuthHandler {
	return &AuthHandler{service: service}
}

// Register handles POST /auth/register: it creates an account and returns the
// user with a fresh token pair.
func (h *AuthHandler) Register(ctx *gin.Context) {
	var req registerRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BindError(ctx, err)
		return
	}

	u, tokens, err := h.service.Register(ctx.Request.Context(), user.User{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}

	response.JSON(ctx, nethttp.StatusCreated, newAuthResponse(u, tokens))
}

// Login handles POST /auth/login: it checks the credentials and returns the
// user with a fresh token pair.
func (h *AuthHandler) Login(ctx *gin.Context) {
	var req loginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BindError(ctx, err)
		return
	}

	u, tokens, err := h.service.Login(ctx.Request.Context(), req.Email, req.Password)
	if err != nil {
		response.Error(ctx, err)
		return
	}

	response.JSON(ctx, nethttp.StatusOK, newAuthResponse(u, tokens))
}

// Refresh handles POST /auth/refresh: it exchanges a refresh token for a new
// token pair.
func (h *AuthHandler) Refresh(ctx *gin.Context) {
	var req refreshRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BindError(ctx, err)
		return
	}

	tokens, err := h.service.Refresh(ctx.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Error(ctx, err)
		return
	}

	response.JSON(ctx, nethttp.StatusOK, refreshResponse{Tokens: newTokensResponse(tokens)})
}

// Logout handles POST /auth/logout: it revokes the given refresh token.
func (h *AuthHandler) Logout(ctx *gin.Context) {
	var req logoutRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BindError(ctx, err)
		return
	}

	if err := h.service.Logout(ctx.Request.Context(), req.RefreshToken); err != nil {
		response.Error(ctx, err)
		return
	}

	response.NoContent(ctx)
}
