package handlers

import (
	"context"
	"errors"
	nethttp "net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/adapters/http/dto"
	"github.com/traily-org/server/internal/domain/photo"
)

type photoService interface {
	Get(ctx context.Context, id string) (photo.Photo, error)
	List(ctx context.Context) ([]photo.Photo, error)
	Create(ctx context.Context, p photo.Photo) (photo.Photo, error)
	Update(ctx context.Context, p photo.Photo) (photo.Photo, error)
	Delete(ctx context.Context, id string) error
}

type PhotoHandler struct {
	service photoService
}

func NewPhotoHandler(service photoService) *PhotoHandler {
	return &PhotoHandler{service: service}
}

func photoErrorStatus(err error) int {
	if errors.Is(err, photo.ErrNotFound) {
		return nethttp.StatusNotFound
	}
	return nethttp.StatusInternalServerError
}

func optionalWKT(s *string) []byte {
	if s == nil {
		return nil
	}
	return []byte(*s)
}

func (h *PhotoHandler) Get(ctx *gin.Context) {
	p, err := h.service.Get(ctx.Request.Context(), ctx.Param("id"))
	if err != nil {
		ctx.JSON(photoErrorStatus(err), gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusOK, dto.NewPhotoResponse(p))
}

func (h *PhotoHandler) List(ctx *gin.Context) {
	list, err := h.service.List(ctx.Request.Context())
	if err != nil {
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	responses := make([]dto.PhotoResponse, len(list))
	for i, p := range list {
		responses[i] = dto.NewPhotoResponse(p)
	}
	ctx.JSON(nethttp.StatusOK, responses)
}

func (h *PhotoHandler) Create(ctx *gin.Context) {
	var req struct {
		UserID     string     `json:"user_id" binding:"required,uuid"`
		StorageKey string     `json:"storage_key" binding:"required"`
		URL        string     `json:"url" binding:"required,url"`
		Location   *string    `json:"location"`
		Visibility string     `json:"visibility" binding:"omitempty,oneof=private friends public"`
		TakenAt    *time.Time `json:"taken_at"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p, err := h.service.Create(ctx.Request.Context(), photo.Photo{
		UserID:     req.UserID,
		StorageKey: req.StorageKey,
		URL:        req.URL,
		Location:   optionalWKT(req.Location),
		Visibility: req.Visibility,
		TakenAt:    req.TakenAt,
	})
	if err != nil {
		ctx.JSON(photoErrorStatus(err), gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusCreated, dto.NewPhotoResponse(p))
}

func (h *PhotoHandler) Update(ctx *gin.Context) {
	var req struct {
		UserID     string     `json:"user_id" binding:"required,uuid"`
		StorageKey string     `json:"storage_key" binding:"required"`
		URL        string     `json:"url" binding:"required,url"`
		Location   *string    `json:"location"`
		Visibility string     `json:"visibility" binding:"required,oneof=private friends public"`
		NSFWStatus string     `json:"nsfw_status" binding:"required,oneof=pending safe flagged rejected"`
		TakenAt    *time.Time `json:"taken_at"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p, err := h.service.Update(ctx.Request.Context(), photo.Photo{
		ID:         ctx.Param("id"),
		UserID:     req.UserID,
		StorageKey: req.StorageKey,
		URL:        req.URL,
		Location:   optionalWKT(req.Location),
		Visibility: req.Visibility,
		NSFWStatus: req.NSFWStatus,
		TakenAt:    req.TakenAt,
	})
	if err != nil {
		ctx.JSON(photoErrorStatus(err), gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusOK, dto.NewPhotoResponse(p))
}

func (h *PhotoHandler) Delete(ctx *gin.Context) {
	if err := h.service.Delete(ctx.Request.Context(), ctx.Param("id")); err != nil {
		ctx.JSON(photoErrorStatus(err), gin.H{"error": err.Error()})
		return
	}
	ctx.Status(nethttp.StatusNoContent)
}
