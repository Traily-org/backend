package handlers

import (
	"context"
	"errors"
	nethttp "net/http"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/domain/waypoint"
)

type waypointService interface {
	Get(ctx context.Context, id string) (waypoint.Waypoint, error)
	List(ctx context.Context) ([]waypoint.Waypoint, error)
	Create(ctx context.Context, e waypoint.Waypoint) (waypoint.Waypoint, error)
	Update(ctx context.Context, e waypoint.Waypoint) (waypoint.Waypoint, error)
	Delete(ctx context.Context, id string) error
}

type WaypointHandler struct {
	service waypointService
}

func NewWaypointHandler(service waypointService) *WaypointHandler {
	return &WaypointHandler{service: service}
}

func (h *WaypointHandler) Get(ctx *gin.Context) {
	e, err := h.service.Get(ctx.Request.Context(), ctx.Param("id"))
	if err != nil {
		writeWaypointError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, e)
}

func (h *WaypointHandler) List(ctx *gin.Context) {
	entities, err := h.service.List(ctx.Request.Context())
	if err != nil {
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusOK, entities)
}

func (h *WaypointHandler) Create(ctx *gin.Context) {
	var e waypoint.Waypoint
	if err := ctx.ShouldBindJSON(&e); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	created, err := h.service.Create(ctx.Request.Context(), e)
	if err != nil {
		writeWaypointError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusCreated, created)
}

func (h *WaypointHandler) Update(ctx *gin.Context) {
	var e waypoint.Waypoint
	if err := ctx.ShouldBindJSON(&e); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	e.ID = ctx.Param("id")

	updated, err := h.service.Update(ctx.Request.Context(), e)
	if err != nil {
		writeWaypointError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, updated)
}

func (h *WaypointHandler) Delete(ctx *gin.Context) {
	if err := h.service.Delete(ctx.Request.Context(), ctx.Param("id")); err != nil {
		writeWaypointError(ctx, err)
		return
	}
	ctx.Status(nethttp.StatusNoContent)
}

func writeWaypointError(ctx *gin.Context, err error) {
	if errors.Is(err, waypoint.ErrNotFound) {
		ctx.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
}
