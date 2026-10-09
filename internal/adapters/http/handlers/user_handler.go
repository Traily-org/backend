package handlers

import (
	"context"
	"errors"
	nethttp "net/http"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/adapters/http/dto"
	"github.com/traily-org/server/internal/domain/user"
)

type userService interface {
	Get(ctx context.Context, id string) (user.User, error)
	Create(ctx context.Context, u user.User) (user.User, error)
	Update(ctx context.Context, u user.User) (user.User, error)
	Delete(ctx context.Context, id string) error
}

type UserHandler struct {
	service userService
}

func NewUserHandler(service userService) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) Get(ctx *gin.Context) {
	id := ctx.Param("id")

	u, err := h.service.Get(ctx.Request.Context(), id)
	if err != nil {
		writeUserError(ctx, err)
		return
	}

	ctx.JSON(nethttp.StatusOK, dto.NewUserResponse(u))
}

func (h *UserHandler) Create(ctx *gin.Context) {
	var req dto.CreateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	u, err := h.service.Create(ctx.Request.Context(), user.User{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	})
	if err != nil {
		writeUserError(ctx, err)
		return
	}

	ctx.JSON(nethttp.StatusCreated, dto.NewUserResponse(u))
}

func (h *UserHandler) Update(ctx *gin.Context) {
	id := ctx.Param("id")

	var req dto.UpdateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	u, err := h.service.Update(ctx.Request.Context(), user.User{
		ID:       id,
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	})
	if err != nil {
		writeUserError(ctx, err)
		return
	}

	ctx.JSON(nethttp.StatusOK, dto.NewUserResponse(u))
}

func (h *UserHandler) Delete(ctx *gin.Context) {
	id := ctx.Param("id")

	if err := h.service.Delete(ctx.Request.Context(), id); err != nil {
		writeUserError(ctx, err)
		return
	}

	ctx.Status(nethttp.StatusNoContent)
}

func writeUserError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, user.ErrNotFound):
		ctx.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, user.ErrEmailTaken):
		ctx.JSON(nethttp.StatusConflict, gin.H{"error": err.Error()})
	default:
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
