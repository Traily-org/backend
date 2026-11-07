package http

import (
	"context"
	nethttp "net/http"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/adapters/http/response"
	"github.com/traily-org/server/internal/domain/trail"
)

// trailService is the subset of the trail domain service used by TrailHandler.
type trailService interface {
	Get(ctx context.Context, id string) (trail.Trail, error)
	ListByUser(ctx context.Context, userID string) ([]trail.Trail, error)
	Create(ctx context.Context, t trail.Trail) (trail.Trail, error)
}

// TrailHandler serves the /trails endpoints.
type TrailHandler struct {
	service trailService
}

// NewTrailHandler returns a TrailHandler backed by service.
func NewTrailHandler(service trailService) *TrailHandler {
	return &TrailHandler{service: service}
}

// Get handles GET /trails/:id.
func (h *TrailHandler) Get(ctx *gin.Context) {
	t, err := h.service.Get(ctx.Request.Context(), ctx.Param("id"))
	if err != nil {
		response.Error(ctx, err)
		return
	}

	response.JSON(ctx, nethttp.StatusOK, newTrailResponse(t))
}

// ListByUser handles GET /trails?user_id=..., returning the trails saved by
// that user, most recent first.
func (h *TrailHandler) ListByUser(ctx *gin.Context) {
	var query listTrailsQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		response.BindError(ctx, err)
		return
	}

	trails, err := h.service.ListByUser(ctx.Request.Context(), query.UserID)
	if err != nil {
		response.Error(ctx, err)
		return
	}

	responses := make([]trailResponse, len(trails))
	for i, t := range trails {
		responses[i] = newTrailResponse(t)
	}

	response.JSON(ctx, nethttp.StatusOK, responses)
}

// Create handles POST /trails: it saves a trail for the user in the payload.
func (h *TrailHandler) Create(ctx *gin.Context) {
	var req createTrailRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BindError(ctx, err)
		return
	}

	t := trail.Trail{
		UserID:                   &req.UserID,
		Name:                     req.Name,
		Description:              req.Description,
		Difficulty:               trail.Difficulty(req.Difficulty),
		DistanceMeters:           *req.DistanceMeters,
		ElevationGainMeters:      *req.ElevationGainMeters,
		ElevationLossMeters:      *req.ElevationLossMeters,
		EstimatedDurationSeconds: req.EstimatedDurationSeconds,
		Path:                     req.Path.path(),
		IsPublished:              true,
	}
	if req.IsPublished != nil {
		t.IsPublished = *req.IsPublished
	}

	created, err := h.service.Create(ctx.Request.Context(), t)
	if err != nil {
		response.Error(ctx, err)
		return
	}

	response.JSON(ctx, nethttp.StatusCreated, newTrailResponse(created))
}
