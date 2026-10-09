package handlers

import (
	"context"
	"errors"
	nethttp "net/http"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/domain/herbierentry"
)

type herbierentryService interface {
	Get(ctx context.Context, id string) (herbierentry.HerbierEntry, error)
	List(ctx context.Context) ([]herbierentry.HerbierEntry, error)
	Create(ctx context.Context, e herbierentry.HerbierEntry) (herbierentry.HerbierEntry, error)
	Update(ctx context.Context, e herbierentry.HerbierEntry) (herbierentry.HerbierEntry, error)
	Delete(ctx context.Context, id string) error
}

type HerbierentryHandler struct {
	service herbierentryService
}

func NewHerbierentryHandler(service herbierentryService) *HerbierentryHandler {
	return &HerbierentryHandler{service: service}
}

func (h *HerbierentryHandler) Get(ctx *gin.Context) {
	id := ctx.Param("id")
	e, err := h.service.Get(ctx.Request.Context(), id)
	if err != nil {
		ctx.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusOK, e)
}

func (h *HerbierentryHandler) List(ctx *gin.Context) {
	entities, err := h.service.List(ctx.Request.Context())
	if err != nil {
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusOK, entities)
}

func (h *HerbierentryHandler) Create(ctx *gin.Context) {
	var e herbierentry.HerbierEntry
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

func (h *HerbierentryHandler) Update(ctx *gin.Context) {
	id := ctx.Param("id")

	var e herbierentry.HerbierEntry
	if err := ctx.ShouldBindJSON(&e); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	e.ID = id

	updated, err := h.service.Update(ctx.Request.Context(), e)
	if err != nil {
		if errors.Is(err, herbierentry.ErrNotFound) {
			ctx.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(nethttp.StatusOK, updated)
}

func (h *HerbierentryHandler) Delete(ctx *gin.Context) {
	id := ctx.Param("id")
	if err := h.service.Delete(ctx.Request.Context(), id); err != nil {
		if errors.Is(err, herbierentry.ErrNotFound) {
			ctx.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.Status(nethttp.StatusNoContent)
}
