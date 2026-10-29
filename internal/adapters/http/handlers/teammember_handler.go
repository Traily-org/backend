package handlers

import (
	"context"
	"errors"
	nethttp "net/http"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/domain/teammember"
)

type teammemberService interface {
	Get(ctx context.Context, id string) (teammember.TeamMember, error)
	List(ctx context.Context) ([]teammember.TeamMember, error)
	Create(ctx context.Context, e teammember.TeamMember) (teammember.TeamMember, error)
	Update(ctx context.Context, e teammember.TeamMember) (teammember.TeamMember, error)
	Delete(ctx context.Context, id string) error
}

type TeamMemberHandler struct {
	service teammemberService
}

func NewTeamMemberHandler(service teammemberService) *TeamMemberHandler {
	return &TeamMemberHandler{service: service}
}

func (h *TeamMemberHandler) Get(ctx *gin.Context) {
	e, err := h.service.Get(ctx.Request.Context(), ctx.Param("id"))
	if err != nil {
		writeTeamMemberError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, e)
}

func (h *TeamMemberHandler) List(ctx *gin.Context) {
	entities, err := h.service.List(ctx.Request.Context())
	if err != nil {
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusOK, entities)
}

func (h *TeamMemberHandler) Create(ctx *gin.Context) {
	var e teammember.TeamMember
	if err := ctx.ShouldBindJSON(&e); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	created, err := h.service.Create(ctx.Request.Context(), e)
	if err != nil {
		writeTeamMemberError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusCreated, created)
}

func (h *TeamMemberHandler) Update(ctx *gin.Context) {
	var e teammember.TeamMember
	if err := ctx.ShouldBindJSON(&e); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	e.ID = ctx.Param("id")

	updated, err := h.service.Update(ctx.Request.Context(), e)
	if err != nil {
		writeTeamMemberError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, updated)
}

func (h *TeamMemberHandler) Delete(ctx *gin.Context) {
	if err := h.service.Delete(ctx.Request.Context(), ctx.Param("id")); err != nil {
		writeTeamMemberError(ctx, err)
		return
	}
	ctx.Status(nethttp.StatusNoContent)
}

func writeTeamMemberError(ctx *gin.Context, err error) {
	if errors.Is(err, teammember.ErrNotFound) {
		ctx.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
}
