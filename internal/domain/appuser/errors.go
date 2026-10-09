package appuser

import "errors"

var (
	ErrNotFound        = errors.New("appuser not found")
	ErrEmailTaken      = errors.New("email already in use")
	ErrUsernameTaken   = errors.New("username already in use")
	ErrInvalidUsername = errors.New("username must be 3-50 characters")
	ErrInvalidEmail    = errors.New("invalid email format")
)
