package report

import "time"

type Report struct {
	ID         string
	ReporterID string
	TargetType string
	TargetID   string
	Reason     string
	Status     string
	ReviewedBy *string
	ReviewedAt *time.Time
	CreatedAt  time.Time
}
