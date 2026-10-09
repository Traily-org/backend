package handlers

import (
	"context"
	"errors"
	nethttp "net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/adapters/http/dto"
	"github.com/traily-org/server/internal/domain/track"
)

type trackService interface {
	Get(ctx context.Context, id string) (track.Track, error)
	ListByUser(ctx context.Context, userID string) ([]track.Track, error)
	Create(ctx context.Context, t track.Track) (track.Track, error)
	Update(ctx context.Context, t track.Track) (track.Track, error)
	Delete(ctx context.Context, id string) error
}

type TrackHandler struct {
	service trackService
}

func NewTrackHandler(service trackService) *TrackHandler {
	return &TrackHandler{service: service}
}

func (h *TrackHandler) Get(ctx *gin.Context) {
	id := ctx.Param("id")
	t, err := h.service.Get(ctx.Request.Context(), id)
	if err != nil {
		ctx.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusOK, dto.NewTrackResponse(t))
}

func (h *TrackHandler) ListByUser(ctx *gin.Context) {
	userID := ctx.Query("user_id")
	if userID == "" {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": "user_id query parameter required"})
		return
	}

	tracks, err := h.service.ListByUser(ctx.Request.Context(), userID)
	if err != nil {
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	responses := make([]dto.TrackResponse, len(tracks))
	for i, t := range tracks {
		responses[i] = dto.NewTrackResponse(t)
	}
	ctx.JSON(nethttp.StatusOK, responses)
}

func (h *TrackHandler) Create(ctx *gin.Context) {
	var req struct {
		UserID         string     `json:"user_id" binding:"required"`
		Route          string     `json:"route" binding:"required"`
		DistanceM      *float64   `json:"distance_m"`
		ElevationGainM *float64   `json:"elevation_gain_m"`
		ElevationLossM *float64   `json:"elevation_loss_m"`
		StartedAt      *time.Time `json:"started_at"`
		EndedAt        *time.Time `json:"ended_at"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	t, err := h.service.Create(ctx.Request.Context(), track.Track{
		UserID:         req.UserID,
		Route:          []byte(req.Route),
		DistanceM:      req.DistanceM,
		ElevationGainM: req.ElevationGainM,
		ElevationLossM: req.ElevationLossM,
		StartedAt:      req.StartedAt,
		EndedAt:        req.EndedAt,
	})
	if err != nil {
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(nethttp.StatusCreated, dto.NewTrackResponse(t))
}

func (h *TrackHandler) Update(ctx *gin.Context) {
	id := ctx.Param("id")

	var req struct {
		UserID         string     `json:"user_id" binding:"required"`
		Route          string     `json:"route" binding:"required"`
		DistanceM      *float64   `json:"distance_m"`
		ElevationGainM *float64   `json:"elevation_gain_m"`
		ElevationLossM *float64   `json:"elevation_loss_m"`
		StartedAt      *time.Time `json:"started_at"`
		EndedAt        *time.Time `json:"ended_at"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	t, err := h.service.Update(ctx.Request.Context(), track.Track{
		ID:             id,
		UserID:         req.UserID,
		Route:          []byte(req.Route),
		DistanceM:      req.DistanceM,
		ElevationGainM: req.ElevationGainM,
		ElevationLossM: req.ElevationLossM,
		StartedAt:      req.StartedAt,
		EndedAt:        req.EndedAt,
	})
	if err != nil {
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(nethttp.StatusOK, dto.NewTrackResponse(t))
}

func (h *TrackHandler) Delete(ctx *gin.Context) {
	id := ctx.Param("id")
	if err := h.service.Delete(ctx.Request.Context(), id); err != nil {
		if errors.Is(err, track.ErrNotFound) {
			ctx.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.Status(nethttp.StatusNoContent)
}
