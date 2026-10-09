package handlers

import (
	"context"
	"errors"
	nethttp "net/http"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/domain/poi"
)

type poiService interface {
	Get(ctx context.Context, id string) (poi.POI, error)
	List(ctx context.Context) ([]poi.POI, error)
	Create(ctx context.Context, e poi.POI) (poi.POI, error)
	Update(ctx context.Context, e poi.POI) (poi.POI, error)
	Delete(ctx context.Context, id string) error
}

type PoiHandler struct {
	service poiService
}

func NewPoiHandler(service poiService) *PoiHandler {
	return &PoiHandler{service: service}
}

func (h *PoiHandler) Get(ctx *gin.Context) {
	id := ctx.Param("id")
	e, err := h.service.Get(ctx.Request.Context(), id)
	if err != nil {
		ctx.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusOK, e)
}

func (h *PoiHandler) List(ctx *gin.Context) {
	entities, err := h.service.List(ctx.Request.Context())
	if err != nil {
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusOK, entities)
}

func (h *PoiHandler) Create(ctx *gin.Context) {
	var e poi.POI
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

func (h *PoiHandler) Update(ctx *gin.Context) {
	id := ctx.Param("id")

	var e poi.POI
	if err := ctx.ShouldBindJSON(&e); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	e.ID = id

	updated, err := h.service.Update(ctx.Request.Context(), e)
	if err != nil {
		if errors.Is(err, poi.ErrNotFound) {
			ctx.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(nethttp.StatusOK, updated)
}

func (h *PoiHandler) Delete(ctx *gin.Context) {
	id := ctx.Param("id")
	if err := h.service.Delete(ctx.Request.Context(), id); err != nil {
		if errors.Is(err, poi.ErrNotFound) {
			ctx.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.Status(nethttp.StatusNoContent)
}
