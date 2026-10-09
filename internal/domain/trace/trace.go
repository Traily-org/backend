package trace

import "time"

type Trace struct {
	ID          string
	UserID      string
	TrackID     string
	Name        string
	Description *string
	Difficulty  *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
