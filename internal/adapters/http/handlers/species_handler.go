package handlers

import (
	"context"
	"errors"
	nethttp "net/http"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/adapters/http/dto"
	"github.com/traily-org/server/internal/domain/species"
)

type speciesService interface {
	Get(ctx context.Context, id string) (species.Species, error)
	List(ctx context.Context) ([]species.Species, error)
	Create(ctx context.Context, s species.Species) (species.Species, error)
	Update(ctx context.Context, s species.Species) (species.Species, error)
	Delete(ctx context.Context, id string) error
}

type speciesRequest struct {
	Category          string  `json:"category" binding:"required,oneof=plant fungus"`
	CommonName        string  `json:"common_name" binding:"required,max=150"`
	ScientificName    *string `json:"scientific_name" binding:"omitempty,max=200"`
	Family            *string `json:"family"`
	Description       *string `json:"description"`
	ProtectionStatus  string  `json:"protection_status" binding:"omitempty,oneof=none concern vulnerable endangered strictly_protected"`
	Rarity            string  `json:"rarity" binding:"omitempty,oneof=common uncommon rare legendary"`
	EthicalAdvice     *string `json:"ethical_advice"`
	ObservationMonths []int32 `json:"observation_months" binding:"omitempty,dive,min=1,max=12"`
	ThumbnailURL      *string `json:"thumbnail_url" binding:"omitempty,url"`
}

func (r speciesRequest) toDomain(id string) species.Species {
	return species.Species{
		ID:                id,
		Category:          species.Category(r.Category),
		CommonName:        r.CommonName,
		ScientificName:    r.ScientificName,
		Family:            r.Family,
		Description:       r.Description,
		ProtectionStatus:  r.ProtectionStatus,
		Rarity:            r.Rarity,
		EthicalAdvice:     r.EthicalAdvice,
		ObservationMonths: r.ObservationMonths,
		ThumbnailURL:      r.ThumbnailURL,
	}
}

type SpeciesHandler struct {
	service speciesService
}

func NewSpeciesHandler(service speciesService) *SpeciesHandler {
	return &SpeciesHandler{service: service}
}

func speciesErrorStatus(err error) int {
	switch {
	case errors.Is(err, species.ErrNotFound):
		return nethttp.StatusNotFound
	case errors.Is(err, species.ErrInvalidCategory), errors.Is(err, species.ErrInvalidMonth):
		return nethttp.StatusBadRequest
	default:
		return nethttp.StatusInternalServerError
	}
}

func (h *SpeciesHandler) Get(ctx *gin.Context) {
	s, err := h.service.Get(ctx.Request.Context(), ctx.Param("id"))
	if err != nil {
		ctx.JSON(speciesErrorStatus(err), gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusOK, dto.NewSpeciesResponse(s))
}

func (h *SpeciesHandler) List(ctx *gin.Context) {
	list, err := h.service.List(ctx.Request.Context())
	if err != nil {
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	responses := make([]dto.SpeciesResponse, len(list))
	for i, s := range list {
		responses[i] = dto.NewSpeciesResponse(s)
	}
	ctx.JSON(nethttp.StatusOK, responses)
}

func (h *SpeciesHandler) Create(ctx *gin.Context) {
	var req speciesRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s, err := h.service.Create(ctx.Request.Context(), req.toDomain(""))
	if err != nil {
		ctx.JSON(speciesErrorStatus(err), gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusCreated, dto.NewSpeciesResponse(s))
}

func (h *SpeciesHandler) Update(ctx *gin.Context) {
	var req speciesRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s, err := h.service.Update(ctx.Request.Context(), req.toDomain(ctx.Param("id")))
	if err != nil {
		ctx.JSON(speciesErrorStatus(err), gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusOK, dto.NewSpeciesResponse(s))
}

func (h *SpeciesHandler) Delete(ctx *gin.Context) {
	if err := h.service.Delete(ctx.Request.Context(), ctx.Param("id")); err != nil {
		ctx.JSON(speciesErrorStatus(err), gin.H{"error": err.Error()})
		return
	}
	ctx.Status(nethttp.StatusNoContent)
}
