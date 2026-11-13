// Package ban holds the ban domain: temporary or permanent bans of an
// account, and the banned identifiers that stop a banned person from signing
// up again after deleting the account.
package ban

import "time"

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

// ReportRef is what the ban domain needs to know about a report. It is a
// reduced view: the report itself, its statuses and its rules belong to the
// report domain.
type ReportRef struct {
	ID string
	// TargetUserID is the reported user, or the author of the reported
	// photo, observation or trail. It is nil when the target no longer
	// exists or has no author.
	TargetUserID *string
	// Dismissed is true when a moderator dismissed the report. A dismissed
	// report cannot justify a ban.
	Dismissed bool
}
