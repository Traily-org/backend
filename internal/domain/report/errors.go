package report

import "github.com/traily-org/server/internal/pkg/apperror"

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
