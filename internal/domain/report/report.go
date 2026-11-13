// Package report holds the report domain: a user flags a photo, an
// observation, a trail or another user to the moderators. Every report keeps
// who reported what, why and when.
package report

import (
	"context"
	"time"

	"github.com/traily-org/server/internal/pkg/apperror"
)

// TargetType is the kind of resource a report is about, stored in
// reports.target_type.
type TargetType string

// Target types allowed by the report_target enum.
const (
	TargetPhoto       TargetType = "photo"
	TargetObservation TargetType = "observation"
	TargetTrail       TargetType = "trail"
	TargetUser        TargetType = "user"
)

// Valid reports whether t is a known target type.
func (t TargetType) Valid() bool {
	switch t {
	case TargetPhoto, TargetObservation, TargetTrail, TargetUser:
		return true
	}
	return false
}

// Status is the review state of a report, stored in reports.status.
type Status string

// Statuses allowed by the report_status enum.
const (
	StatusPending   Status = "pending"
	StatusReviewed  Status = "reviewed"
	StatusDismissed Status = "dismissed"
)

// Valid reports whether s is a known status.
func (s Status) Valid() bool {
	switch s {
	case StatusPending, StatusReviewed, StatusDismissed:
		return true
	}
	return false
}

// MaxReasonLength is the maximum length of a reason, in characters.
const MaxReasonLength = 1000

// Pagination bounds of List.
const (
	DefaultLimit = 20
	MaxLimit     = 100
)

// Report is a report about a target. Traceability: ReporterID, TargetType,
// TargetID, Reason and CreatedAt are set on every report.
type Report struct {
	ID         string
	ReporterID string
	TargetType TargetType
	TargetID   string
	Reason     string
	Status     Status
	// ReviewedBy and ReviewedAt are set once a moderator handled the report.
	ReviewedBy *string
	ReviewedAt *time.Time
	CreatedAt  time.Time
}

// Input describes a report to create.
type Input struct {
	ReporterID string
	TargetType TargetType
	TargetID   string
	Reason     string
}

// Filter selects the reports returned by List. A nil field matches all.
type Filter struct {
	Status     *Status
	TargetType *TargetType
	// Limit defaults to DefaultLimit and is capped at MaxLimit.
	Limit  int
	Offset int
}

// Page is a page of reports, newest first, with the total number of reports
// matching the filter.
type Page struct {
	Items  []Report
	Total  int
	Limit  int
	Offset int
}

// Repository persists reports.
type Repository interface {
	// TargetAuthor returns the author of the target as seen by reporterID: a
	// nil author means the target exists but has none. It returns
	// ErrTargetNotFound when the target does not exist or is not visible to
	// reporterID (private or unpublished content of someone else).
	TargetAuthor(ctx context.Context, kind TargetType, id, reporterID string) (*string, error)
	// Create stores r with status pending and returns it, or
	// ErrAlreadyReported when reporterID already has a pending report on the
	// same target.
	Create(ctx context.Context, r Report) (Report, error)
	// List returns the reports matching f, newest first.
	List(ctx context.Context, f Filter) (Page, error)
}

// Errors returned by the report domain.
var (
	// ErrReasonRequired is returned when a report has no reason.
	ErrReasonRequired = apperror.Validation("reason is required")
	// ErrReasonTooLong is returned when the reason exceeds MaxReasonLength.
	ErrReasonTooLong = apperror.Validation("reason must be at most 1000 characters")
	// ErrInvalidTargetType is returned for an unknown target type.
	ErrInvalidTargetType = apperror.Validation("target_type must be one of: photo, observation, trail, user")
	// ErrInvalidStatus is returned for an unknown status filter.
	ErrInvalidStatus = apperror.Validation("status must be one of: pending, reviewed, dismissed")
	// ErrTargetNotFound is returned when the target does not exist or cannot
	// be seen by the reporter.
	ErrTargetNotFound = apperror.NotFound("reported target not found")
	// ErrCannotReportSelf is returned when a user reports themselves or
	// content they authored.
	ErrCannotReportSelf = apperror.Validation("you cannot report yourself or your own content")
	// ErrAlreadyReported is returned when the reporter already has a pending
	// report on the target.
	ErrAlreadyReported = apperror.Conflict("you already have a pending report on this target")
)
