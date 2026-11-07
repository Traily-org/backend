package trail

import "github.com/traily-org/server/internal/pkg/apperror"

// Sentinel errors returned by the trail domain. They are compared by identity
// with errors.Is and carry their own API error code.
var (
	// ErrNotFound is returned when no trail matches the requested identifier.
	ErrNotFound = apperror.NotFound("trail not found")
	// ErrUserRequired is returned when a trail is created without an owner.
	ErrUserRequired = apperror.Validation("user_id is required")
	// ErrNameRequired is returned when a trail has a blank name.
	ErrNameRequired = apperror.Validation("name is required")
	// ErrInvalidDifficulty is returned for a difficulty that is not supported.
	ErrInvalidDifficulty = apperror.Validation("difficulty must be one of: easy, moderate, hard, expert")
	// ErrInvalidDistance is returned for a negative distance.
	ErrInvalidDistance = apperror.Validation("distance_meters must not be negative")
	// ErrInvalidElevation is returned for a negative elevation gain or loss.
	ErrInvalidElevation = apperror.Validation("elevation gain and loss must not be negative")
	// ErrInvalidDuration is returned for a negative estimated duration.
	ErrInvalidDuration = apperror.Validation("estimated_duration_seconds must not be negative")
	// ErrPathTooShort is returned when the path has fewer than MinPathPoints points.
	ErrPathTooShort = apperror.Validation("path must contain at least 2 points")
	// ErrInvalidCoordinate is returned for a longitude outside [-180, 180] or a
	// latitude outside [-90, 90].
	ErrInvalidCoordinate = apperror.Validation("path coordinates must be within longitude [-180, 180] and latitude [-90, 90]")
)
