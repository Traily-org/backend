package trail

import "time"

type Trail struct {
	ID                 string
	UserID             *string
	Name               string
	Description        *string
	Difficulty         string
	DistanceM          float64
	ElevationGainM     float64
	ElevationLossM     float64
	EstimatedDurationS int
	Path               []byte
	IsOfficial         bool
	IsPublished        bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
