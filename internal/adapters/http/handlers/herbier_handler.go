package handlers

import (
	"context"
	"errors"
	nethttp "net/http"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/domain/herbier"
)

type herbierService interface {
	Get(ctx context.Context, id string) (herbier.Herbier, error)
	List(ctx context.Context) ([]herbier.Herbier, error)
	Create(ctx context.Context, e herbier.Herbier) (herbier.Herbier, error)
	Update(ctx context.Context, e herbier.Herbier) (herbier.Herbier, error)
	Delete(ctx context.Context, id string) error
}

type HerbierHandler struct {
	service herbierService
}

func NewHerbierHandler(service herbierService) *HerbierHandler {
	return &HerbierHandler{service: service}
}

func (h *HerbierHandler) Get(ctx *gin.Context) {
	id := ctx.Param("id")
	e, err := h.service.Get(ctx.Request.Context(), id)
	if err != nil {
		ctx.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusOK, e)
}

func (h *HerbierHandler) List(ctx *gin.Context) {
	entities, err := h.service.List(ctx.Request.Context())
	if err != nil {
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusOK, entities)
}

func (h *HerbierHandler) Create(ctx *gin.Context) {
	var e herbier.Herbier
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

func (h *HerbierHandler) Update(ctx *gin.Context) {
	id := ctx.Param("id")

	var e herbier.Herbier
	if err := ctx.ShouldBindJSON(&e); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	e.ID = id

	updated, err := h.service.Update(ctx.Request.Context(), e)
	if err != nil {
		if errors.Is(err, herbier.ErrNotFound) {
			ctx.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(nethttp.StatusOK, updated)
}

func (h *HerbierHandler) Delete(ctx *gin.Context) {
	id := ctx.Param("id")
	if err := h.service.Delete(ctx.Request.Context(), id); err != nil {
		if errors.Is(err, herbier.ErrNotFound) {
			ctx.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.Status(nethttp.StatusNoContent)
}
