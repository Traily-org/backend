package ban

import (
	"context"
	"time"
)

// Repository persists bans. Implementations return ErrNotBanned when a user
// has no active ban and ErrReportNotFound when a report does not exist.
type Repository interface {
	// Report returns the report identified by id, or ErrReportNotFound.
	Report(ctx context.Context, id string) (ReportRef, error)
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
