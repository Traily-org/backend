package handlers

import (
	"context"
	"errors"
	nethttp "net/http"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/domain/badge"
)

type badgeService interface {
	Get(ctx context.Context, id string) (badge.Badge, error)
	List(ctx context.Context) ([]badge.Badge, error)
	Create(ctx context.Context, e badge.Badge) (badge.Badge, error)
	Update(ctx context.Context, e badge.Badge) (badge.Badge, error)
	Delete(ctx context.Context, id string) error
}

type BadgeHandler struct {
	service badgeService
}

func NewBadgeHandler(service badgeService) *BadgeHandler {
	return &BadgeHandler{service: service}
}

func (h *BadgeHandler) Get(ctx *gin.Context) {
	e, err := h.service.Get(ctx.Request.Context(), ctx.Param("id"))
	if err != nil {
		writeBadgeError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, e)
}

func (h *BadgeHandler) List(ctx *gin.Context) {
	entities, err := h.service.List(ctx.Request.Context())
	if err != nil {
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusOK, entities)
}

func (h *BadgeHandler) Create(ctx *gin.Context) {
	var e badge.Badge
	if err := ctx.ShouldBindJSON(&e); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	created, err := h.service.Create(ctx.Request.Context(), e)
	if err != nil {
		writeBadgeError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusCreated, created)
}

func (h *BadgeHandler) Update(ctx *gin.Context) {
	var e badge.Badge
	if err := ctx.ShouldBindJSON(&e); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	e.ID = ctx.Param("id")

	updated, err := h.service.Update(ctx.Request.Context(), e)
	if err != nil {
		writeBadgeError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, updated)
}

func (h *BadgeHandler) Delete(ctx *gin.Context) {
	if err := h.service.Delete(ctx.Request.Context(), ctx.Param("id")); err != nil {
		writeBadgeError(ctx, err)
		return
	}
	ctx.Status(nethttp.StatusNoContent)
}

func writeBadgeError(ctx *gin.Context, err error) {
	if errors.Is(err, badge.ErrNotFound) {
		ctx.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
}
