package observation

import "time"

type Observation struct {
	ID               string
	UserID           string
	SpeciesID        *string
	PhotoID          *string
	TraceID          *string
	ActivityID       *string
	Location         []byte // WKT
	Altitude         *float64
	ConfidenceScore  *float64
	ValidationStatus string
	Visibility       string
	ObservedAt       time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
