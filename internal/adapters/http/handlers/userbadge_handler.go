package handlers

import (
	"context"
	"errors"
	nethttp "net/http"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/domain/userbadge"
)

type userbadgeService interface {
	Get(ctx context.Context, id string) (userbadge.UserBadge, error)
	List(ctx context.Context) ([]userbadge.UserBadge, error)
	Create(ctx context.Context, e userbadge.UserBadge) (userbadge.UserBadge, error)
	Update(ctx context.Context, e userbadge.UserBadge) (userbadge.UserBadge, error)
	Delete(ctx context.Context, id string) error
}

type UserBadgeHandler struct {
	service userbadgeService
}

func NewUserBadgeHandler(service userbadgeService) *UserBadgeHandler {
	return &UserBadgeHandler{service: service}
}

func (h *UserBadgeHandler) Get(ctx *gin.Context) {
	e, err := h.service.Get(ctx.Request.Context(), ctx.Param("id"))
	if err != nil {
		writeUserBadgeError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, e)
}

func (h *UserBadgeHandler) List(ctx *gin.Context) {
	entities, err := h.service.List(ctx.Request.Context())
	if err != nil {
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusOK, entities)
}

func (h *UserBadgeHandler) Create(ctx *gin.Context) {
	var e userbadge.UserBadge
	if err := ctx.ShouldBindJSON(&e); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	created, err := h.service.Create(ctx.Request.Context(), e)
	if err != nil {
		writeUserBadgeError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusCreated, created)
}

func (h *UserBadgeHandler) Update(ctx *gin.Context) {
	var e userbadge.UserBadge
	if err := ctx.ShouldBindJSON(&e); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	e.ID = ctx.Param("id")

	updated, err := h.service.Update(ctx.Request.Context(), e)
	if err != nil {
		writeUserBadgeError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, updated)
}

func (h *UserBadgeHandler) Delete(ctx *gin.Context) {
	if err := h.service.Delete(ctx.Request.Context(), ctx.Param("id")); err != nil {
		writeUserBadgeError(ctx, err)
		return
	}
	ctx.Status(nethttp.StatusNoContent)
}

func writeUserBadgeError(ctx *gin.Context, err error) {
	if errors.Is(err, userbadge.ErrNotFound) {
		ctx.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
}
