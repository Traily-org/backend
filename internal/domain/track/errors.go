package track

import "errors"

var (
	ErrNotFound          = errors.New("track not found")
	ErrInvalidDistance   = errors.New("distance must be positive")
	ErrInvalidElevation  = errors.New("elevation must be positive")
	ErrInvalidDates      = errors.New("ended_at must be after or equal to started_at")
)
