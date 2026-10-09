package handlers

import (
	"context"
	"errors"
	nethttp "net/http"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/adapters/http/dto"
	"github.com/traily-org/server/internal/domain/appuser"
)

type appUserService interface {
	Get(ctx context.Context, id string) (appuser.AppUser, error)
	GetByUsername(ctx context.Context, username string) (appuser.AppUser, error)
	List(ctx context.Context) ([]appuser.AppUser, error)
	Create(ctx context.Context, u appuser.AppUser) (appuser.AppUser, error)
	Update(ctx context.Context, u appuser.AppUser) (appuser.AppUser, error)
	Delete(ctx context.Context, id string) error
}

type AppUserHandler struct {
	service appUserService
}

func NewAppUserHandler(service appUserService) *AppUserHandler {
	return &AppUserHandler{service: service}
}

func (h *AppUserHandler) Get(ctx *gin.Context) {
	id := ctx.Param("id")
	u, err := h.service.Get(ctx.Request.Context(), id)
	if err != nil {
		writeAppUserError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, dto.NewAppUserResponse(u))
}

func (h *AppUserHandler) List(ctx *gin.Context) {
	users, err := h.service.List(ctx.Request.Context())
	if err != nil {
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	responses := make([]dto.AppUserResponse, len(users))
	for i, u := range users {
		responses[i] = dto.NewAppUserResponse(u)
	}
	ctx.JSON(nethttp.StatusOK, responses)
}

func (h *AppUserHandler) Create(ctx *gin.Context) {
	var req dto.CreateAppUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	u, err := h.service.Create(ctx.Request.Context(), appuser.AppUser{
		Username:    req.Username,
		Email:       req.Email,
		DisplayName: req.DisplayName,
		AvatarURL:   req.AvatarURL,
	})
	if err != nil {
		writeAppUserError(ctx, err)
		return
	}

	ctx.JSON(nethttp.StatusCreated, dto.NewAppUserResponse(u))
}

func (h *AppUserHandler) Update(ctx *gin.Context) {
	id := ctx.Param("id")

	var req dto.UpdateAppUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	u, err := h.service.Update(ctx.Request.Context(), appuser.AppUser{
		ID:          id,
		Username:    req.Username,
		Email:       req.Email,
		DisplayName: req.DisplayName,
		AvatarURL:   req.AvatarURL,
	})
	if err != nil {
		writeAppUserError(ctx, err)
		return
	}

	ctx.JSON(nethttp.StatusOK, dto.NewAppUserResponse(u))
}

func (h *AppUserHandler) Delete(ctx *gin.Context) {
	id := ctx.Param("id")
	if err := h.service.Delete(ctx.Request.Context(), id); err != nil {
		writeAppUserError(ctx, err)
		return
	}
	ctx.Status(nethttp.StatusNoContent)
}

func writeAppUserError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, appuser.ErrNotFound):
		ctx.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, appuser.ErrEmailTaken), errors.Is(err, appuser.ErrUsernameTaken):
		ctx.JSON(nethttp.StatusConflict, gin.H{"error": err.Error()})
	default:
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
