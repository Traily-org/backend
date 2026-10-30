package http

import (
	"context"
	nethttp "net/http"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/adapters/http/middleware"
	"github.com/traily-org/server/internal/adapters/http/response"
	"github.com/traily-org/server/internal/domain/user"
	"github.com/traily-org/server/internal/pkg/apperror"
)

// errAuthenticationRequired is returned when a /users/me endpoint is reached
// without an authenticated user in the request context.
var errAuthenticationRequired = apperror.Unauthorized("authentication required")

// userService is the subset of the user domain service used by UserHandler.
type userService interface {
	Get(ctx context.Context, id string) (user.User, error)
	UpdateProfile(ctx context.Context, u user.User, currentPassword string) (user.User, error)
	Delete(ctx context.Context, id string) error
}

// UserHandler serves the /users endpoints. Every endpoint expects the
// authentication middleware to have run before it.
type UserHandler struct {
	service userService
}

// NewUserHandler returns a UserHandler backed by service.
func NewUserHandler(service userService) *UserHandler {
	return &UserHandler{service: service}
}

// Get handles GET /users/:id. It returns the public profile of the user,
// without private fields such as the email or role.
func (h *UserHandler) Get(c *gin.Context) {
	u, err := h.service.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.JSON(c, nethttp.StatusOK, newPublicUserResponse(u))
}

// GetMe handles GET /users/me. It returns the full profile of the
// authenticated user.
func (h *UserHandler) GetMe(c *gin.Context) {
	id, ok := middleware.UserID(c)
	if !ok {
		response.Error(c, errAuthenticationRequired)
		return
	}

	u, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.JSON(c, nethttp.StatusOK, newUserResponse(u))
}

// UpdateMe handles PUT /users/me. It replaces the profile of the
// authenticated user; an empty password keeps the current one. Changing the
// email or the password requires current_password.
func (h *UserHandler) UpdateMe(c *gin.Context) {
	id, ok := middleware.UserID(c)
	if !ok {
		response.Error(c, errAuthenticationRequired)
		return
	}

	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}

	u, err := h.service.UpdateProfile(c.Request.Context(), user.User{
		ID:          id,
		Email:       req.Email,
		Password:    req.Password,
		Name:        req.Name,
		Username:    optional(req.Username),
		DisplayName: optional(req.DisplayName),
		AvatarURL:   optional(req.AvatarURL),
	}, req.CurrentPassword)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.JSON(c, nethttp.StatusOK, newUserResponse(u))
}

// DeleteMe handles DELETE /users/me. It deletes the authenticated user.
func (h *UserHandler) DeleteMe(c *gin.Context) {
	id, ok := middleware.UserID(c)
	if !ok {
		response.Error(c, errAuthenticationRequired)
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}

	response.NoContent(c)
}
