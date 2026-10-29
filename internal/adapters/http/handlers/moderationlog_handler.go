package handlers

import (
	"context"
	"errors"
	nethttp "net/http"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/domain/moderationlog"
)

type moderationlogService interface {
	Get(ctx context.Context, id string) (moderationlog.ModerationLog, error)
	List(ctx context.Context) ([]moderationlog.ModerationLog, error)
	Create(ctx context.Context, e moderationlog.ModerationLog) (moderationlog.ModerationLog, error)
	Update(ctx context.Context, e moderationlog.ModerationLog) (moderationlog.ModerationLog, error)
	Delete(ctx context.Context, id string) error
}

type ModerationLogHandler struct {
	service moderationlogService
}

func NewModerationLogHandler(service moderationlogService) *ModerationLogHandler {
	return &ModerationLogHandler{service: service}
}

func (h *ModerationLogHandler) Get(ctx *gin.Context) {
	e, err := h.service.Get(ctx.Request.Context(), ctx.Param("id"))
	if err != nil {
		writeModerationLogError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, e)
}

func (h *ModerationLogHandler) List(ctx *gin.Context) {
	entities, err := h.service.List(ctx.Request.Context())
	if err != nil {
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusOK, entities)
}

func (h *ModerationLogHandler) Create(ctx *gin.Context) {
	var e moderationlog.ModerationLog
	if err := ctx.ShouldBindJSON(&e); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	created, err := h.service.Create(ctx.Request.Context(), e)
	if err != nil {
		writeModerationLogError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusCreated, created)
}

func (h *ModerationLogHandler) Update(ctx *gin.Context) {
	var e moderationlog.ModerationLog
	if err := ctx.ShouldBindJSON(&e); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	e.ID = ctx.Param("id")

	updated, err := h.service.Update(ctx.Request.Context(), e)
	if err != nil {
		writeModerationLogError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, updated)
}

func (h *ModerationLogHandler) Delete(ctx *gin.Context) {
	if err := h.service.Delete(ctx.Request.Context(), ctx.Param("id")); err != nil {
		writeModerationLogError(ctx, err)
		return
	}
	ctx.Status(nethttp.StatusNoContent)
}

func writeModerationLogError(ctx *gin.Context, err error) {
	if errors.Is(err, moderationlog.ErrNotFound) {
		ctx.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
}
