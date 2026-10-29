package handlers

import (
	"context"
	"errors"
	nethttp "net/http"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/domain/species"
)

type speciesService interface {
	Get(ctx context.Context, id string) (species.Species, error)
	List(ctx context.Context) ([]species.Species, error)
	Create(ctx context.Context, e species.Species) (species.Species, error)
	Update(ctx context.Context, e species.Species) (species.Species, error)
	Delete(ctx context.Context, id string) error
}

type SpeciesHandler struct {
	service speciesService
}

func NewSpeciesHandler(service speciesService) *SpeciesHandler {
	return &SpeciesHandler{service: service}
}

func (h *SpeciesHandler) Get(ctx *gin.Context) {
	e, err := h.service.Get(ctx.Request.Context(), ctx.Param("id"))
	if err != nil {
		writeSpeciesError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, e)
}

func (h *SpeciesHandler) List(ctx *gin.Context) {
	entities, err := h.service.List(ctx.Request.Context())
	if err != nil {
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusOK, entities)
}

func (h *SpeciesHandler) Create(ctx *gin.Context) {
	var e species.Species
	if err := ctx.ShouldBindJSON(&e); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	created, err := h.service.Create(ctx.Request.Context(), e)
	if err != nil {
		writeSpeciesError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusCreated, created)
}

func (h *SpeciesHandler) Update(ctx *gin.Context) {
	var e species.Species
	if err := ctx.ShouldBindJSON(&e); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	e.ID = ctx.Param("id")

	updated, err := h.service.Update(ctx.Request.Context(), e)
	if err != nil {
		writeSpeciesError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, updated)
}

func (h *SpeciesHandler) Delete(ctx *gin.Context) {
	if err := h.service.Delete(ctx.Request.Context(), ctx.Param("id")); err != nil {
		writeSpeciesError(ctx, err)
		return
	}
	ctx.Status(nethttp.StatusNoContent)
}

func writeSpeciesError(ctx *gin.Context, err error) {
	if errors.Is(err, species.ErrNotFound) {
		ctx.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
}
