package http

import (
	"context"
	nethttp "net/http"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/adapters/http/response"
	"github.com/traily-org/server/internal/domain/user"
)

// userService is the subset of the user domain service used by UserHandler.
type userService interface {
	Get(ctx context.Context, id string) (user.User, error)
	Create(ctx context.Context, u user.User) (user.User, error)
	Update(ctx context.Context, u user.User) (user.User, error)
	Delete(ctx context.Context, id string) error
}

// UserHandler serves the /users endpoints.
type UserHandler struct {
	service userService
}

// NewUserHandler returns a UserHandler backed by service.
func NewUserHandler(service userService) *UserHandler {
	return &UserHandler{service: service}
}

// Get handles GET /users/:id.
func (h *UserHandler) Get(ctx *gin.Context) {
	u, err := h.service.Get(ctx.Request.Context(), ctx.Param("id"))
	if err != nil {
		response.Error(ctx, err)
		return
	}

	response.JSON(ctx, nethttp.StatusOK, newUserResponse(u))
}

// Create handles POST /users.
func (h *UserHandler) Create(ctx *gin.Context) {
	var req createUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BindError(ctx, err)
		return
	}

	u, err := h.service.Create(ctx.Request.Context(), user.User{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}

	response.JSON(ctx, nethttp.StatusCreated, newUserResponse(u))
}

// Update handles PUT /users/:id. An empty password keeps the current one.
func (h *UserHandler) Update(ctx *gin.Context) {
	var req updateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BindError(ctx, err)
		return
	}

	u, err := h.service.Update(ctx.Request.Context(), user.User{
		ID:       ctx.Param("id"),
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}

	response.JSON(ctx, nethttp.StatusOK, newUserResponse(u))
}

// Delete handles DELETE /users/:id.
func (h *UserHandler) Delete(ctx *gin.Context) {
	if err := h.service.Delete(ctx.Request.Context(), ctx.Param("id")); err != nil {
		response.Error(ctx, err)
		return
	}

	response.NoContent(ctx)
}
