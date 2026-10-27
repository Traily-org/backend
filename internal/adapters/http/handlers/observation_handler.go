package handlers

import (
	"context"
	"errors"
	nethttp "net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/adapters/http/dto"
	"github.com/traily-org/server/internal/domain/observation"
)

type observationService interface {
	Get(ctx context.Context, id string) (observation.Observation, error)
	List(ctx context.Context) ([]observation.Observation, error)
	Create(ctx context.Context, o observation.Observation) (observation.Observation, error)
	Update(ctx context.Context, o observation.Observation) (observation.Observation, error)
	Delete(ctx context.Context, id string) error
}

type ObservationHandler struct {
	service observationService
}

func NewObservationHandler(service observationService) *ObservationHandler {
	return &ObservationHandler{service: service}
}

func observationErrorStatus(err error) int {
	switch {
	case errors.Is(err, observation.ErrNotFound):
		return nethttp.StatusNotFound
	case errors.Is(err, observation.ErrMissingLocation), errors.Is(err, observation.ErrInvalidConfidence):
		return nethttp.StatusBadRequest
	default:
		return nethttp.StatusInternalServerError
	}
}

func (h *ObservationHandler) Get(ctx *gin.Context) {
	o, err := h.service.Get(ctx.Request.Context(), ctx.Param("id"))
	if err != nil {
		ctx.JSON(observationErrorStatus(err), gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusOK, dto.NewObservationResponse(o))
}

func (h *ObservationHandler) List(ctx *gin.Context) {
	list, err := h.service.List(ctx.Request.Context())
	if err != nil {
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	responses := make([]dto.ObservationResponse, len(list))
	for i, o := range list {
		responses[i] = dto.NewObservationResponse(o)
	}
	ctx.JSON(nethttp.StatusOK, responses)
}

func (h *ObservationHandler) Create(ctx *gin.Context) {
	var req struct {
		UserID          string     `json:"user_id" binding:"required,uuid"`
		SpeciesID       *string    `json:"species_id" binding:"omitempty,uuid"`
		PhotoID         *string    `json:"photo_id" binding:"omitempty,uuid"`
		TraceID         *string    `json:"trace_id" binding:"omitempty,uuid"`
		ActivityID      *string    `json:"activity_id" binding:"omitempty,uuid"`
		Location        string     `json:"location" binding:"required"`
		Altitude        *float64   `json:"altitude"`
		ConfidenceScore *float64   `json:"confidence_score"`
		Visibility      string     `json:"visibility" binding:"omitempty,oneof=private friends public"`
		ObservedAt      *time.Time `json:"observed_at"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	o := observation.Observation{
		UserID:          req.UserID,
		SpeciesID:       req.SpeciesID,
		PhotoID:         req.PhotoID,
		TraceID:         req.TraceID,
		ActivityID:      req.ActivityID,
		Location:        []byte(req.Location),
		Altitude:        req.Altitude,
		ConfidenceScore: req.ConfidenceScore,
		Visibility:      req.Visibility,
	}
	if req.ObservedAt != nil {
		o.ObservedAt = *req.ObservedAt
	}
	created, err := h.service.Create(ctx.Request.Context(), o)
	if err != nil {
		ctx.JSON(observationErrorStatus(err), gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusCreated, dto.NewObservationResponse(created))
}

func (h *ObservationHandler) Update(ctx *gin.Context) {
	var req struct {
		UserID           string    `json:"user_id" binding:"required,uuid"`
		SpeciesID        *string   `json:"species_id" binding:"omitempty,uuid"`
		PhotoID          *string   `json:"photo_id" binding:"omitempty,uuid"`
		TraceID          *string   `json:"trace_id" binding:"omitempty,uuid"`
		ActivityID       *string   `json:"activity_id" binding:"omitempty,uuid"`
		Location         string    `json:"location" binding:"required"`
		Altitude         *float64  `json:"altitude"`
		ConfidenceScore  *float64  `json:"confidence_score"`
		ValidationStatus string    `json:"validation_status" binding:"required,oneof=pending validated rejected"`
		Visibility       string    `json:"visibility" binding:"required,oneof=private friends public"`
		ObservedAt       time.Time `json:"observed_at" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updated, err := h.service.Update(ctx.Request.Context(), observation.Observation{
		ID:               ctx.Param("id"),
		UserID:           req.UserID,
		SpeciesID:        req.SpeciesID,
		PhotoID:          req.PhotoID,
		TraceID:          req.TraceID,
		ActivityID:       req.ActivityID,
		Location:         []byte(req.Location),
		Altitude:         req.Altitude,
		ConfidenceScore:  req.ConfidenceScore,
		ValidationStatus: req.ValidationStatus,
		Visibility:       req.Visibility,
		ObservedAt:       req.ObservedAt,
	})
	if err != nil {
		ctx.JSON(observationErrorStatus(err), gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusOK, dto.NewObservationResponse(updated))
}

func (h *ObservationHandler) Delete(ctx *gin.Context) {
	if err := h.service.Delete(ctx.Request.Context(), ctx.Param("id")); err != nil {
		ctx.JSON(observationErrorStatus(err), gin.H{"error": err.Error()})
		return
	}
	ctx.Status(nethttp.StatusNoContent)
}
