package handlers

import (
	"context"
	"errors"
	nethttp "net/http"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/domain/team"
)

type teamService interface {
	Get(ctx context.Context, id string) (team.Team, error)
	List(ctx context.Context) ([]team.Team, error)
	Create(ctx context.Context, e team.Team) (team.Team, error)
	Update(ctx context.Context, e team.Team) (team.Team, error)
	Delete(ctx context.Context, id string) error
}

type TeamHandler struct {
	service teamService
}

func NewTeamHandler(service teamService) *TeamHandler {
	return &TeamHandler{service: service}
}

func (h *TeamHandler) Get(ctx *gin.Context) {
	e, err := h.service.Get(ctx.Request.Context(), ctx.Param("id"))
	if err != nil {
		writeTeamError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, e)
}

func (h *TeamHandler) List(ctx *gin.Context) {
	entities, err := h.service.List(ctx.Request.Context())
	if err != nil {
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusOK, entities)
}

func (h *TeamHandler) Create(ctx *gin.Context) {
	var e team.Team
	if err := ctx.ShouldBindJSON(&e); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	created, err := h.service.Create(ctx.Request.Context(), e)
	if err != nil {
		writeTeamError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusCreated, created)
}

func (h *TeamHandler) Update(ctx *gin.Context) {
	var e team.Team
	if err := ctx.ShouldBindJSON(&e); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	e.ID = ctx.Param("id")

	updated, err := h.service.Update(ctx.Request.Context(), e)
	if err != nil {
		writeTeamError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, updated)
}

func (h *TeamHandler) Delete(ctx *gin.Context) {
	if err := h.service.Delete(ctx.Request.Context(), ctx.Param("id")); err != nil {
		writeTeamError(ctx, err)
		return
	}
	ctx.Status(nethttp.StatusNoContent)
}

func writeTeamError(ctx *gin.Context, err error) {
	if errors.Is(err, team.ErrNotFound) {
		ctx.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
}
