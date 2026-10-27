package species

import "errors"

var (
	ErrNotFound        = errors.New("species not found")
	ErrInvalidCategory = errors.New("category must be plant or fungus")
	ErrInvalidMonth    = errors.New("observation months must be between 1 and 12")
)
