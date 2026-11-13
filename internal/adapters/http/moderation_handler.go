package http

import (
	"context"
	nethttp "net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/adapters/http/middleware"
	"github.com/traily-org/server/internal/adapters/http/response"
	"github.com/traily-org/server/internal/domain/ban"
)

// banService is the subset of the ban service used by ModerationHandler.
type banService interface {
	Ban(ctx context.Context, in ban.Input) (ban.Ban, error)
	Lift(ctx context.Context, userID string, by ban.Moderator) error
	Active(ctx context.Context, userID string) (ban.Ban, error)
}

// ModerationHandler serves the /moderation endpoints. Every endpoint expects
// the authentication and moderator role middlewares to have run before it.
type ModerationHandler struct {
	service banService
}

// NewModerationHandler returns a ModerationHandler backed by service.
func NewModerationHandler(service banService) *ModerationHandler {
	return &ModerationHandler{service: service}
}

// BanUser handles POST /moderation/users/:id/ban. It bans the user until
// expires_at, or permanently when expires_at is omitted, on the ground of the
// report report_id, which is then marked as reviewed.
func (h *ModerationHandler) BanUser(c *gin.Context) {
	by, ok := moderator(c)
	if !ok {
		response.Error(c, errAuthenticationRequired)
		return
	}

	var req banRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}

	b, err := h.service.Ban(c.Request.Context(), ban.Input{
		UserID:    c.Param("id"),
		ReportID:  req.ReportID,
		Reason:    req.Reason,
		ExpiresAt: req.ExpiresAt,
		By:        by,
	})
	if err != nil {
		response.Error(c, err)
		return
	}

	response.JSON(c, nethttp.StatusCreated, newBanResponse(b))
}

// GetBan handles GET /moderation/users/:id/ban. It returns the active ban of
// the user that ends last, or 404 NOT_FOUND when the user is not banned.
func (h *ModerationHandler) GetBan(c *gin.Context) {
	b, err := h.service.Active(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.JSON(c, nethttp.StatusOK, newBanResponse(b))
}

// LiftBan handles DELETE /moderation/users/:id/ban. It ends every active ban
// of the user, or answers 404 NOT_FOUND when the user is not banned.
func (h *ModerationHandler) LiftBan(c *gin.Context) {
	by, ok := moderator(c)
	if !ok {
		response.Error(c, errAuthenticationRequired)
		return
	}

	if err := h.service.Lift(c.Request.Context(), c.Param("id"), by); err != nil {
		response.Error(c, err)
		return
	}

	response.NoContent(c)
}

// moderator returns the authenticated user from the request context.
func moderator(c *gin.Context) (ban.Moderator, bool) {
	id, ok := middleware.UserID(c)
	if !ok {
		return ban.Moderator{}, false
	}
	role, ok := middleware.UserRole(c)
	if !ok {
		return ban.Moderator{}, false
	}
	return ban.Moderator{ID: id, Role: role}, true
}

// banRequest is the payload of POST /moderation/users/:id/ban.
type banRequest struct {
	// ReportID is the report about the user, or about content they
	// authored, that justifies the ban.
	ReportID string `json:"report_id" binding:"required,uuid"`
	Reason   string `json:"reason" binding:"required,max=1000"`
	// ExpiresAt is the end of the ban in RFC 3339; omitted or null makes it
	// permanent.
	ExpiresAt *time.Time `json:"expires_at"`
}

// banResponse is the representation of a ban. Dates are UTC; a null
// expires_at means a permanent ban.
type banResponse struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	ReportID  *string    `json:"report_id"`
	Reason    string     `json:"reason"`
	BannedBy  *string    `json:"banned_by"`
	ExpiresAt *time.Time `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
}

func newBanResponse(b ban.Ban) banResponse {
	return banResponse{
		ID:        b.ID,
		UserID:    b.UserID,
		ReportID:  b.ReportID,
		Reason:    b.Reason,
		BannedBy:  b.BannedBy,
		ExpiresAt: utcPtr(b.ExpiresAt),
		CreatedAt: b.CreatedAt.UTC(),
	}
}

// utcPtr returns t converted to UTC, or nil.
func utcPtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}
