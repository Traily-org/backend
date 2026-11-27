package ban

import (
	"errors"
	"time"

	"github.com/traily-org/server/internal/pkg/apperror"
)

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
