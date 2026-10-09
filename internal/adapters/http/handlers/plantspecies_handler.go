package handlers

import (
	"context"
	"errors"
	nethttp "net/http"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/domain/plantspecies"
)

type plantspeciesService interface {
	Get(ctx context.Context, id string) (plantspecies.PlantSpecies, error)
	List(ctx context.Context) ([]plantspecies.PlantSpecies, error)
	Create(ctx context.Context, e plantspecies.PlantSpecies) (plantspecies.PlantSpecies, error)
	Update(ctx context.Context, e plantspecies.PlantSpecies) (plantspecies.PlantSpecies, error)
	Delete(ctx context.Context, id string) error
}

type PlantspeciesHandler struct {
	service plantspeciesService
}

func NewPlantspeciesHandler(service plantspeciesService) *PlantspeciesHandler {
	return &PlantspeciesHandler{service: service}
}

func (h *PlantspeciesHandler) Get(ctx *gin.Context) {
	id := ctx.Param("id")
	e, err := h.service.Get(ctx.Request.Context(), id)
	if err != nil {
		ctx.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusOK, e)
}

func (h *PlantspeciesHandler) List(ctx *gin.Context) {
	entities, err := h.service.List(ctx.Request.Context())
	if err != nil {
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusOK, entities)
}

func (h *PlantspeciesHandler) Create(ctx *gin.Context) {
	var e plantspecies.PlantSpecies
	if err := ctx.ShouldBindJSON(&e); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	created, err := h.service.Create(ctx.Request.Context(), e)
	if err != nil {
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(nethttp.StatusCreated, created)
}

func (h *PlantspeciesHandler) Update(ctx *gin.Context) {
	id := ctx.Param("id")

	var e plantspecies.PlantSpecies
	if err := ctx.ShouldBindJSON(&e); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	e.ID = id

	updated, err := h.service.Update(ctx.Request.Context(), e)
	if err != nil {
		if errors.Is(err, plantspecies.ErrNotFound) {
			ctx.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(nethttp.StatusOK, updated)
}

func (h *PlantspeciesHandler) Delete(ctx *gin.Context) {
	id := ctx.Param("id")
	if err := h.service.Delete(ctx.Request.Context(), id); err != nil {
		if errors.Is(err, plantspecies.ErrNotFound) {
			ctx.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.Status(nethttp.StatusNoContent)
}
