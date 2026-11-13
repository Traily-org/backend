// Package ban holds the ban domain: temporary or permanent bans of an
// account, and the banned identifiers that stop a banned person from signing
// up again after deleting the account.
package ban

import (
	"context"
	"errors"
	"time"

	"github.com/traily-org/server/internal/pkg/apperror"
)

// Ban is a ban of an account. It is active from CreatedAt until ExpiresAt, or
// forever when ExpiresAt is nil, unless it was lifted earlier.
type Ban struct {
	ID     string
	UserID string
	// ReportID is the report that led to the ban. It is always set on
	// creation and only becomes nil if the report is deleted later.
	ReportID *string
	Reason   string
	// BannedBy is the moderator who banned, or nil once their account is
	// deleted.
	BannedBy *string
	// ExpiresAt is the end of the ban, or nil for a permanent ban.
	ExpiresAt *time.Time
	// LiftedAt is set when a moderator ended the ban before ExpiresAt.
	LiftedAt *time.Time
	// LiftedBy is the moderator who lifted the ban, when known.
	LiftedBy  *string
	CreatedAt time.Time
}

// ActiveAt reports whether the ban is in force at t.
func (b Ban) ActiveAt(t time.Time) bool {
	return b.LiftedAt == nil && (b.ExpiresAt == nil || t.Before(*b.ExpiresAt))
}

// IdentifierType is the kind of a banned identifier.
type IdentifierType string

// IdentifierEmail is a banned email address, hashed with HashEmail.
const IdentifierEmail IdentifierType = "email"

// Identifier is a hashed value that cannot be used to sign up while it is
// banned. It outlives the account it was taken from.
type Identifier struct {
	Type      IdentifierType
	ValueHash string
	// ExpiresAt is copied from the ban; nil means permanent.
	ExpiresAt *time.Time
}

// ReportStatus is the review state of a report, stored in reports.status.
type ReportStatus string

// Report statuses allowed by the report_status enum.
const (
	ReportPending   ReportStatus = "pending"
	ReportReviewed  ReportStatus = "reviewed"
	ReportDismissed ReportStatus = "dismissed"
)

// Report is what the ban domain needs to know about a report.
type Report struct {
	ID string
	// TargetUserID is the reported user, or the author of the reported
	// photo, observation or trail. It is nil when the target no longer
	// exists or has no author.
	TargetUserID *string
	Status       ReportStatus
}

// Repository persists bans. Implementations return ErrNotBanned when a user
// has no active ban and ErrReportNotFound when a report does not exist.
type Repository interface {
	// Report returns the report identified by id, or ErrReportNotFound.
	Report(ctx context.Context, id string) (Report, error)
	// Create stores b, bans the identifiers, marks the report b.ReportID as
	// reviewed by b.BannedBy and logs the ban in moderation_logs, in one
	// transaction. An identifier that is already banned keeps the latest of
	// the two expiries. b.BannedBy and b.ReportID must be set.
	Create(ctx context.Context, b Ban, ids []Identifier) (Ban, error)
	// Active returns the ban of userID active at at that ends last, or
	// ErrNotBanned.
	Active(ctx context.Context, userID string, at time.Time) (Ban, error)
	// Lift lifts every ban of userID active at at, removes their banned
	// identifiers and logs it in moderation_logs, in one transaction. It
	// returns the number of bans lifted.
	Lift(ctx context.Context, userID, liftedBy string, at time.Time) (int, error)
	// IdentifierBanned reports whether the identifier is banned at at.
	IdentifierBanned(ctx context.Context, kind IdentifierType, hash string, at time.Time) (bool, error)
}

// ErrBanned is wrapped by every error that refuses access because of a ban,
// so that callers can test for it with errors.Is.
var ErrBanned = errors.New("banned")

// Sentinel errors returned by the ban domain.
var (
	// ErrNotBanned is returned when the user has no active ban.
	ErrNotBanned = apperror.NotFound("user is not banned")
	// ErrCannotBanSelf is returned when a moderator tries to ban themselves.
	ErrCannotBanSelf = apperror.Validation("you cannot ban yourself")
	// ErrCannotBanStaff is returned when a moderator tries to ban another
	// moderator or an admin. Only admins can.
	ErrCannotBanStaff = apperror.Forbidden("only an admin can ban a moderator or an admin")
	// ErrExpiryNotInFuture is returned when a ban would end before it starts.
	ErrExpiryNotInFuture = apperror.Validation("expires_at must be in the future")
	// ErrReasonRequired is returned when a ban has no reason.
	ErrReasonRequired = apperror.Validation("reason is required")
	// ErrReportRequired is returned when a ban is not tied to a report.
	ErrReportRequired = apperror.Validation("report_id is required")
	// ErrReportNotFound is returned when the report does not exist.
	ErrReportNotFound = apperror.NotFound("report not found")
	// ErrReportNotAboutUser is returned when the report concerns neither the
	// user nor content they authored.
	ErrReportNotAboutUser = apperror.Validation("report does not concern this user")
	// ErrReportDismissed is returned when the report was dismissed.
	ErrReportDismissed = apperror.Conflict("report was dismissed")
)

// bannedAccountError returns the error given to a banned user. Its message
// tells until when, in UTC RFC 3339, so that clients can show it.
func bannedAccountError(b Ban) error {
	msg := "account permanently banned"
	if b.ExpiresAt != nil {
		msg = "account banned until " + b.ExpiresAt.UTC().Format(time.RFC3339)
	}
	return &apperror.Error{Code: apperror.CodeAccountBanned, Message: msg, Err: ErrBanned}
}

// errBannedEmail is given when signing up with a banned email address.
var errBannedEmail = &apperror.Error{
	Code:    apperror.CodeAccountBanned,
	Message: "this email address is banned",
	Err:     ErrBanned,
}
