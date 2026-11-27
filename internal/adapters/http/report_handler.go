package http

import (
	"context"
	nethttp "net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/adapters/http/middleware"
	"github.com/traily-org/server/internal/adapters/http/response"
	"github.com/traily-org/server/internal/domain/report"
)

// reportService is the subset of the report service used by ReportHandler.
type reportService interface {
	Create(ctx context.Context, in report.Input) (report.Report, error)
	List(ctx context.Context, f report.Filter) (report.Page, error)
}

// ReportHandler serves POST /reports, open to every signed-in user, and
// GET /moderation/reports, which the server restricts to moderators.
type ReportHandler struct {
	service reportService
}

// NewReportHandler returns a ReportHandler backed by service.
func NewReportHandler(service reportService) *ReportHandler {
	return &ReportHandler{service: service}
}

// Create handles POST /reports. It records a report from the authenticated
// user about a photo, an observation, a trail or a user.
func (h *ReportHandler) Create(c *gin.Context) {
	reporterID, ok := middleware.UserID(c)
	if !ok {
		response.Error(c, errAuthenticationRequired)
		return
	}

	var req createReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}

	created, err := h.service.Create(c.Request.Context(), report.Input{
		ReporterID: reporterID,
		TargetType: report.TargetType(req.TargetType),
		TargetID:   req.TargetID,
		Reason:     req.Reason,
	})
	if err != nil {
		response.Error(c, err)
		return
	}

	response.JSON(c, nethttp.StatusCreated, newReportResponse(created))
}

// List handles GET /moderation/reports, the moderation dashboard feed. Reports
// come newest first and can be filtered by status and target_type.
func (h *ReportHandler) List(c *gin.Context) {
	var q listReportsQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.BindError(c, err)
		return
	}

	var filter report.Filter
	if q.Limit != nil {
		filter.Limit = *q.Limit
	}
	if q.Offset != nil {
		filter.Offset = *q.Offset
	}
	if q.Status != "" {
		s := report.Status(q.Status)
		filter.Status = &s
	}
	if q.TargetType != "" {
		t := report.TargetType(q.TargetType)
		filter.TargetType = &t
	}

	page, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.JSON(c, nethttp.StatusOK, newReportListResponse(page))
}

// createReportRequest is the payload of POST /reports.
type createReportRequest struct {
	TargetType string `json:"target_type" binding:"required,oneof=photo observation trail user"`
	TargetID   string `json:"target_id" binding:"required,uuid"`
	Reason     string `json:"reason" binding:"required,max=1000"`
}

// listReportsQuery is the query string of GET /moderation/reports.
type listReportsQuery struct {
	Status     string `form:"status" binding:"omitempty,oneof=pending reviewed dismissed"`
	TargetType string `form:"target_type" binding:"omitempty,oneof=photo observation trail user"`
	// Pointers tell an omitted value from an explicit 0, which limit rejects.
	Limit  *int `form:"limit" binding:"omitempty,min=1,max=100"`
	Offset *int `form:"offset" binding:"omitempty,min=0"`
}

// reportResponse is the representation of a report. Dates are UTC.
type reportResponse struct {
	ID         string     `json:"id"`
	ReporterID string     `json:"reporter_id"`
	TargetType string     `json:"target_type"`
	TargetID   string     `json:"target_id"`
	Reason     string     `json:"reason"`
	Status     string     `json:"status"`
	ReviewedBy *string    `json:"reviewed_by"`
	ReviewedAt *time.Time `json:"reviewed_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

func newReportResponse(r report.Report) reportResponse {
	return reportResponse{
		ID:         r.ID,
		ReporterID: r.ReporterID,
		TargetType: string(r.TargetType),
		TargetID:   r.TargetID,
		Reason:     r.Reason,
		Status:     string(r.Status),
		ReviewedBy: r.ReviewedBy,
		ReviewedAt: utcPtr(r.ReviewedAt),
		CreatedAt:  r.CreatedAt.UTC(),
	}
}

// reportListResponse is a page of reports with the total number of matches.
type reportListResponse struct {
	Items  []reportResponse `json:"items"`
	Total  int              `json:"total"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
}

func newReportListResponse(p report.Page) reportListResponse {
	items := make([]reportResponse, 0, len(p.Items))
	for _, r := range p.Items {
		items = append(items, newReportResponse(r))
	}
	return reportListResponse{Items: items, Total: p.Total, Limit: p.Limit, Offset: p.Offset}
}
