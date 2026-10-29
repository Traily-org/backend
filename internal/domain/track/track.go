package track

import (
	"time"
)

type Track struct {
	ID             string
	UserID         string
	TrailID        *string
	Route          []byte
	DistanceM      *float64
	ElevationGainM *float64
	ElevationLossM *float64
	StartedAt      *time.Time
	EndedAt        *time.Time
	IsPrivate      bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
