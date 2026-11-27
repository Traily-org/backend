// Package report holds the report domain: a user flags a photo, an
// observation, a trail or another user to the moderators. Every report keeps
// who reported what, why and when.
package report

import "time"

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
