package handlers

import (
	"context"
	"errors"
	nethttp "net/http"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/domain/report"
)

type reportService interface {
	Get(ctx context.Context, id string) (report.Report, error)
	List(ctx context.Context) ([]report.Report, error)
	Create(ctx context.Context, e report.Report) (report.Report, error)
	Update(ctx context.Context, e report.Report) (report.Report, error)
	Delete(ctx context.Context, id string) error
}

type ReportHandler struct {
	service reportService
}

func NewReportHandler(service reportService) *ReportHandler {
	return &ReportHandler{service: service}
}

func (h *ReportHandler) Get(ctx *gin.Context) {
	e, err := h.service.Get(ctx.Request.Context(), ctx.Param("id"))
	if err != nil {
		writeReportError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, e)
}

func (h *ReportHandler) List(ctx *gin.Context) {
	entities, err := h.service.List(ctx.Request.Context())
	if err != nil {
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusOK, entities)
}

func (h *ReportHandler) Create(ctx *gin.Context) {
	var e report.Report
	if err := ctx.ShouldBindJSON(&e); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	created, err := h.service.Create(ctx.Request.Context(), e)
	if err != nil {
		writeReportError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusCreated, created)
}

func (h *ReportHandler) Update(ctx *gin.Context) {
	var e report.Report
	if err := ctx.ShouldBindJSON(&e); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	e.ID = ctx.Param("id")

	updated, err := h.service.Update(ctx.Request.Context(), e)
	if err != nil {
		writeReportError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, updated)
}

func (h *ReportHandler) Delete(ctx *gin.Context) {
	if err := h.service.Delete(ctx.Request.Context(), ctx.Param("id")); err != nil {
		writeReportError(ctx, err)
		return
	}
	ctx.Status(nethttp.StatusNoContent)
}

func writeReportError(ctx *gin.Context, err error) {
	if errors.Is(err, report.ErrNotFound) {
		ctx.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
}
