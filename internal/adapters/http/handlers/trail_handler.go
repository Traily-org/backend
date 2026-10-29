package handlers

import (
	"context"
	"errors"
	nethttp "net/http"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/domain/trail"
)

type trailService interface {
	Get(ctx context.Context, id string) (trail.Trail, error)
	List(ctx context.Context) ([]trail.Trail, error)
	Create(ctx context.Context, e trail.Trail) (trail.Trail, error)
	Update(ctx context.Context, e trail.Trail) (trail.Trail, error)
	Delete(ctx context.Context, id string) error
}

type TrailHandler struct {
	service trailService
}

func NewTrailHandler(service trailService) *TrailHandler {
	return &TrailHandler{service: service}
}

func (h *TrailHandler) Get(ctx *gin.Context) {
	e, err := h.service.Get(ctx.Request.Context(), ctx.Param("id"))
	if err != nil {
		writeTrailError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, e)
}

func (h *TrailHandler) List(ctx *gin.Context) {
	entities, err := h.service.List(ctx.Request.Context())
	if err != nil {
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusOK, entities)
}

func (h *TrailHandler) Create(ctx *gin.Context) {
	var e trail.Trail
	if err := ctx.ShouldBindJSON(&e); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	created, err := h.service.Create(ctx.Request.Context(), e)
	if err != nil {
		writeTrailError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusCreated, created)
}

func (h *TrailHandler) Update(ctx *gin.Context) {
	var e trail.Trail
	if err := ctx.ShouldBindJSON(&e); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	e.ID = ctx.Param("id")

	updated, err := h.service.Update(ctx.Request.Context(), e)
	if err != nil {
		writeTrailError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, updated)
}

func (h *TrailHandler) Delete(ctx *gin.Context) {
	if err := h.service.Delete(ctx.Request.Context(), ctx.Param("id")); err != nil {
		writeTrailError(ctx, err)
		return
	}
	ctx.Status(nethttp.StatusNoContent)
}

func writeTrailError(ctx *gin.Context, err error) {
	if errors.Is(err, trail.ErrNotFound) {
		ctx.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
}
