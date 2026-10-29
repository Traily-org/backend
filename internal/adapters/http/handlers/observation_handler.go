package handlers

import (
	"context"
	"errors"
	nethttp "net/http"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/domain/observation"
)

type observationService interface {
	Get(ctx context.Context, id string) (observation.Observation, error)
	List(ctx context.Context) ([]observation.Observation, error)
	Create(ctx context.Context, e observation.Observation) (observation.Observation, error)
	Update(ctx context.Context, e observation.Observation) (observation.Observation, error)
	Delete(ctx context.Context, id string) error
}

type ObservationHandler struct {
	service observationService
}

func NewObservationHandler(service observationService) *ObservationHandler {
	return &ObservationHandler{service: service}
}

func (h *ObservationHandler) Get(ctx *gin.Context) {
	e, err := h.service.Get(ctx.Request.Context(), ctx.Param("id"))
	if err != nil {
		writeObservationError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, e)
}

func (h *ObservationHandler) List(ctx *gin.Context) {
	entities, err := h.service.List(ctx.Request.Context())
	if err != nil {
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusOK, entities)
}

func (h *ObservationHandler) Create(ctx *gin.Context) {
	var e observation.Observation
	if err := ctx.ShouldBindJSON(&e); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	created, err := h.service.Create(ctx.Request.Context(), e)
	if err != nil {
		writeObservationError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusCreated, created)
}

func (h *ObservationHandler) Update(ctx *gin.Context) {
	var e observation.Observation
	if err := ctx.ShouldBindJSON(&e); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	e.ID = ctx.Param("id")

	updated, err := h.service.Update(ctx.Request.Context(), e)
	if err != nil {
		writeObservationError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, updated)
}

func (h *ObservationHandler) Delete(ctx *gin.Context) {
	if err := h.service.Delete(ctx.Request.Context(), ctx.Param("id")); err != nil {
		writeObservationError(ctx, err)
		return
	}
	ctx.Status(nethttp.StatusNoContent)
}

func writeObservationError(ctx *gin.Context, err error) {
	if errors.Is(err, observation.ErrNotFound) {
		ctx.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
}
