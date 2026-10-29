package handlers

import (
	"context"
	"errors"
	nethttp "net/http"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/domain/refreshtoken"
)

type refreshtokenService interface {
	Get(ctx context.Context, id string) (refreshtoken.RefreshToken, error)
	List(ctx context.Context) ([]refreshtoken.RefreshToken, error)
	Create(ctx context.Context, e refreshtoken.RefreshToken) (refreshtoken.RefreshToken, error)
	Update(ctx context.Context, e refreshtoken.RefreshToken) (refreshtoken.RefreshToken, error)
	Delete(ctx context.Context, id string) error
}

type RefreshTokenHandler struct {
	service refreshtokenService
}

func NewRefreshTokenHandler(service refreshtokenService) *RefreshTokenHandler {
	return &RefreshTokenHandler{service: service}
}

func (h *RefreshTokenHandler) Get(ctx *gin.Context) {
	e, err := h.service.Get(ctx.Request.Context(), ctx.Param("id"))
	if err != nil {
		writeRefreshTokenError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, e)
}

func (h *RefreshTokenHandler) List(ctx *gin.Context) {
	entities, err := h.service.List(ctx.Request.Context())
	if err != nil {
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusOK, entities)
}

func (h *RefreshTokenHandler) Create(ctx *gin.Context) {
	var e refreshtoken.RefreshToken
	if err := ctx.ShouldBindJSON(&e); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	created, err := h.service.Create(ctx.Request.Context(), e)
	if err != nil {
		writeRefreshTokenError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusCreated, created)
}

func (h *RefreshTokenHandler) Update(ctx *gin.Context) {
	var e refreshtoken.RefreshToken
	if err := ctx.ShouldBindJSON(&e); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	e.ID = ctx.Param("id")

	updated, err := h.service.Update(ctx.Request.Context(), e)
	if err != nil {
		writeRefreshTokenError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, updated)
}

func (h *RefreshTokenHandler) Delete(ctx *gin.Context) {
	if err := h.service.Delete(ctx.Request.Context(), ctx.Param("id")); err != nil {
		writeRefreshTokenError(ctx, err)
		return
	}
	ctx.Status(nethttp.StatusNoContent)
}

func writeRefreshTokenError(ctx *gin.Context, err error) {
	if errors.Is(err, refreshtoken.ErrNotFound) {
		ctx.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
}
