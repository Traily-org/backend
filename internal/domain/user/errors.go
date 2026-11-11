package user

import "github.com/traily-org/server/internal/pkg/apperror"

// Sentinel errors returned by the user domain. They are compared by identity
// with errors.Is and carry their own API error code.
var (
	// ErrNotFound is returned when no user matches the requested identifier.
	ErrNotFound = apperror.NotFound("user not found")
	// ErrEmailTaken is returned when the email is already used by another user.
	ErrEmailTaken = apperror.Conflict("email already taken")
	// ErrUsernameTaken is returned when the username is already used by
	// another user.
	ErrUsernameTaken = apperror.Conflict("username already taken")
	// ErrInvalidCredentials is returned by Service.Authenticate when the email
	// is unknown or the password does not match. Both cases share this error
	// so that callers cannot tell whether an account exists.
	ErrInvalidCredentials = apperror.Unauthorized("invalid email or password")
	// ErrPasswordRequired is returned when a user is created without a password.
	ErrPasswordRequired = apperror.Validation("password is required")
	// ErrPasswordTooLong is returned when a password exceeds MaxPasswordBytes.
	ErrPasswordTooLong = apperror.Validation("password must be at most 72 bytes")
	// ErrCurrentPasswordRequired is returned by Service.UpdateProfile when the
	// email or the password changes without the current password.
	ErrCurrentPasswordRequired = apperror.Validation("current_password is required to change email or password")
	// ErrWrongCurrentPassword is returned by Service.UpdateProfile when the
	// current password does not match.
	ErrWrongCurrentPassword = apperror.Forbidden("current password is incorrect")
)
