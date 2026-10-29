package observation

import "time"

type Observation struct {
	ID               string
	UserID           string
	SpeciesID        *string
	PhotoID          *string
	TrailID          *string
	TrackID          *string
	Location         []byte
	Altitude         *float64
	ConfidenceScore  *float64
	ValidationStatus string
	Visibility       string
	ObservedAt       *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
