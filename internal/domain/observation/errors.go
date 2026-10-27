package observation

import "errors"

var (
	ErrNotFound          = errors.New("observation not found")
	ErrMissingLocation   = errors.New("location is required")
	ErrInvalidConfidence = errors.New("confidence score must be between 0 and 1")
)
