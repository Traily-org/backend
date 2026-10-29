package user

import "github.com/traily-org/server/internal/pkg/apperror"

// Sentinel errors returned by the user domain. They are compared by identity
// with errors.Is and carry their own API error code.
var (
	// ErrNotFound is returned when no user matches the requested identifier.
	ErrNotFound = apperror.NotFound("user not found")
	// ErrEmailTaken is returned when the email is already used by another user.
	ErrEmailTaken = apperror.Conflict("email already taken")
)
