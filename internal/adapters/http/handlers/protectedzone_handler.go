package handlers

import (
	"context"
	"errors"
	nethttp "net/http"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/domain/protectedzone"
)

type protectedzoneService interface {
	Get(ctx context.Context, id string) (protectedzone.ProtectedZone, error)
	List(ctx context.Context) ([]protectedzone.ProtectedZone, error)
	Create(ctx context.Context, e protectedzone.ProtectedZone) (protectedzone.ProtectedZone, error)
	Update(ctx context.Context, e protectedzone.ProtectedZone) (protectedzone.ProtectedZone, error)
	Delete(ctx context.Context, id string) error
}

type ProtectedZoneHandler struct {
	service protectedzoneService
}

func NewProtectedZoneHandler(service protectedzoneService) *ProtectedZoneHandler {
	return &ProtectedZoneHandler{service: service}
}

func (h *ProtectedZoneHandler) Get(ctx *gin.Context) {
	e, err := h.service.Get(ctx.Request.Context(), ctx.Param("id"))
	if err != nil {
		writeProtectedZoneError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, e)
}

func (h *ProtectedZoneHandler) List(ctx *gin.Context) {
	entities, err := h.service.List(ctx.Request.Context())
	if err != nil {
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusOK, entities)
}

func (h *ProtectedZoneHandler) Create(ctx *gin.Context) {
	var e protectedzone.ProtectedZone
	if err := ctx.ShouldBindJSON(&e); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	created, err := h.service.Create(ctx.Request.Context(), e)
	if err != nil {
		writeProtectedZoneError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusCreated, created)
}

func (h *ProtectedZoneHandler) Update(ctx *gin.Context) {
	var e protectedzone.ProtectedZone
	if err := ctx.ShouldBindJSON(&e); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	e.ID = ctx.Param("id")

	updated, err := h.service.Update(ctx.Request.Context(), e)
	if err != nil {
		writeProtectedZoneError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, updated)
}

func (h *ProtectedZoneHandler) Delete(ctx *gin.Context) {
	if err := h.service.Delete(ctx.Request.Context(), ctx.Param("id")); err != nil {
		writeProtectedZoneError(ctx, err)
		return
	}
	ctx.Status(nethttp.StatusNoContent)
}

func writeProtectedZoneError(ctx *gin.Context, err error) {
	if errors.Is(err, protectedzone.ErrNotFound) {
		ctx.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
}
