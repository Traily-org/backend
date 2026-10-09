package handlers

import (
	"context"
	"errors"
	nethttp "net/http"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/domain/trace"
)

type traceService interface {
	Get(ctx context.Context, id string) (trace.Trace, error)
	List(ctx context.Context) ([]trace.Trace, error)
	Create(ctx context.Context, e trace.Trace) (trace.Trace, error)
	Update(ctx context.Context, e trace.Trace) (trace.Trace, error)
	Delete(ctx context.Context, id string) error
}

type TraceHandler struct {
	service traceService
}

func NewTraceHandler(service traceService) *TraceHandler {
	return &TraceHandler{service: service}
}

func (h *TraceHandler) Get(ctx *gin.Context) {
	id := ctx.Param("id")
	e, err := h.service.Get(ctx.Request.Context(), id)
	if err != nil {
		ctx.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusOK, e)
}

func (h *TraceHandler) List(ctx *gin.Context) {
	entities, err := h.service.List(ctx.Request.Context())
	if err != nil {
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusOK, entities)
}

func (h *TraceHandler) Create(ctx *gin.Context) {
	var e trace.Trace
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

func (h *TraceHandler) Update(ctx *gin.Context) {
	id := ctx.Param("id")

	var e trace.Trace
	if err := ctx.ShouldBindJSON(&e); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	e.ID = id

	updated, err := h.service.Update(ctx.Request.Context(), e)
	if err != nil {
		if errors.Is(err, trace.ErrNotFound) {
			ctx.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(nethttp.StatusOK, updated)
}

func (h *TraceHandler) Delete(ctx *gin.Context) {
	id := ctx.Param("id")
	if err := h.service.Delete(ctx.Request.Context(), id); err != nil {
		if errors.Is(err, trace.ErrNotFound) {
			ctx.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.Status(nethttp.StatusNoContent)
}
