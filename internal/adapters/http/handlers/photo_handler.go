package handlers

import (
	"context"
	"errors"
	nethttp "net/http"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/domain/photo"
)

type photoService interface {
	Get(ctx context.Context, id string) (photo.Photo, error)
	List(ctx context.Context) ([]photo.Photo, error)
	Create(ctx context.Context, e photo.Photo) (photo.Photo, error)
	Update(ctx context.Context, e photo.Photo) (photo.Photo, error)
	Delete(ctx context.Context, id string) error
}

type PhotoHandler struct {
	service photoService
}

func NewPhotoHandler(service photoService) *PhotoHandler {
	return &PhotoHandler{service: service}
}

func (h *PhotoHandler) Get(ctx *gin.Context) {
	e, err := h.service.Get(ctx.Request.Context(), ctx.Param("id"))
	if err != nil {
		writePhotoError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, e)
}

func (h *PhotoHandler) List(ctx *gin.Context) {
	entities, err := h.service.List(ctx.Request.Context())
	if err != nil {
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusOK, entities)
}

func (h *PhotoHandler) Create(ctx *gin.Context) {
	var e photo.Photo
	if err := ctx.ShouldBindJSON(&e); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	created, err := h.service.Create(ctx.Request.Context(), e)
	if err != nil {
		writePhotoError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusCreated, created)
}

func (h *PhotoHandler) Update(ctx *gin.Context) {
	var e photo.Photo
	if err := ctx.ShouldBindJSON(&e); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	e.ID = ctx.Param("id")

	updated, err := h.service.Update(ctx.Request.Context(), e)
	if err != nil {
		writePhotoError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, updated)
}

func (h *PhotoHandler) Delete(ctx *gin.Context) {
	if err := h.service.Delete(ctx.Request.Context(), ctx.Param("id")); err != nil {
		writePhotoError(ctx, err)
		return
	}
	ctx.Status(nethttp.StatusNoContent)
}

func writePhotoError(ctx *gin.Context, err error) {
	if errors.Is(err, photo.ErrNotFound) {
		ctx.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
}
