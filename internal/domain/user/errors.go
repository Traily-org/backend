package user

import "github.com/traily-org/server/internal/pkg/apperror"

// Sentinel errors returned by the user domain. They are compared by identity
// with errors.Is and carry their own API error code.
var (
	// ErrNotFound is returned when no user matches the requested identifier.
	ErrNotFound = apperror.NotFound("user not found")
	// ErrEmailTaken is returned when the email is already used by another user.
	ErrEmailTaken = apperror.Conflict("email already taken")
	// ErrPasswordRequired is returned when a user is created without a password.
	ErrPasswordRequired = apperror.Validation("password is required")
	// ErrPasswordTooLong is returned when a password exceeds MaxPasswordBytes.
	ErrPasswordTooLong = apperror.Validation("password must be at most 72 bytes")
)
