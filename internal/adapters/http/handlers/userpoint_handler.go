package handlers

import (
	"context"
	"errors"
	nethttp "net/http"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/domain/userpoint"
)

type userpointService interface {
	Get(ctx context.Context, id string) (userpoint.UserPoint, error)
	List(ctx context.Context) ([]userpoint.UserPoint, error)
	Create(ctx context.Context, e userpoint.UserPoint) (userpoint.UserPoint, error)
	Update(ctx context.Context, e userpoint.UserPoint) (userpoint.UserPoint, error)
	Delete(ctx context.Context, id string) error
}

type UserPointHandler struct {
	service userpointService
}

func NewUserPointHandler(service userpointService) *UserPointHandler {
	return &UserPointHandler{service: service}
}

func (h *UserPointHandler) Get(ctx *gin.Context) {
	e, err := h.service.Get(ctx.Request.Context(), ctx.Param("id"))
	if err != nil {
		writeUserPointError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, e)
}

func (h *UserPointHandler) List(ctx *gin.Context) {
	entities, err := h.service.List(ctx.Request.Context())
	if err != nil {
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusOK, entities)
}

func (h *UserPointHandler) Create(ctx *gin.Context) {
	var e userpoint.UserPoint
	if err := ctx.ShouldBindJSON(&e); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	created, err := h.service.Create(ctx.Request.Context(), e)
	if err != nil {
		writeUserPointError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusCreated, created)
}

func (h *UserPointHandler) Update(ctx *gin.Context) {
	var e userpoint.UserPoint
	if err := ctx.ShouldBindJSON(&e); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	e.ID = ctx.Param("id")

	updated, err := h.service.Update(ctx.Request.Context(), e)
	if err != nil {
		writeUserPointError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, updated)
}

func (h *UserPointHandler) Delete(ctx *gin.Context) {
	if err := h.service.Delete(ctx.Request.Context(), ctx.Param("id")); err != nil {
		writeUserPointError(ctx, err)
		return
	}
	ctx.Status(nethttp.StatusNoContent)
}

func writeUserPointError(ctx *gin.Context, err error) {
	if errors.Is(err, userpoint.ErrNotFound) {
		ctx.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
}
