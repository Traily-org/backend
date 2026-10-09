package track

import (
	"time"
)

type Track struct {
	ID              string
	UserID          string
	Route           []byte
	DistanceM       *float64
	ElevationGainM  *float64
	ElevationLossM  *float64
	StartedAt       *time.Time
	EndedAt         *time.Time
	CreatedAt       time.Time
}
