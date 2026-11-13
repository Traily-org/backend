package report

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
