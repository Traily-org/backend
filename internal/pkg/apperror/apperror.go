// Package apperror defines the application error type shared by every layer
// of the server. Each error carries a stable machine-readable Code, a message
// that is safe to expose to API clients and an optional underlying cause.
package apperror

import (
	nethttp "net/http"
)

// Code is a stable, machine-readable error identifier exposed to API clients.
type Code string

// Error codes exposed to API clients.
const (
	// CodeValidation reports an invalid input (payload, parameter, value).
	CodeValidation Code = "VALIDATION_ERROR"
	// CodeUnauthorized reports a missing or invalid authentication.
	CodeUnauthorized Code = "UNAUTHORIZED"
	// CodeForbidden reports an authenticated caller lacking permissions.
	CodeForbidden Code = "FORBIDDEN"
	// CodeNotFound reports a resource that does not exist.
	CodeNotFound Code = "NOT_FOUND"
	// CodeConflict reports a conflict with the current state of a resource.
	CodeConflict Code = "CONFLICT"
	// CodeRateLimited reports a caller that exceeded its rate limit.
	CodeRateLimited Code = "RATE_LIMITED"
	// CodeInternal reports an unexpected server-side failure.
	CodeInternal Code = "INTERNAL_ERROR"
)

// internalMessage is the only message ever exposed for internal errors.
const internalMessage = "internal server error"

// Error is an application error. Message is safe to return to API clients;
// Err holds the underlying cause, which must never be exposed to them.
type Error struct {
	// Code is the machine-readable error code.
	Code Code
	// Message is a human-readable message safe to expose to clients.
	Message string
	// Err is the optional underlying cause, kept for logging and unwrapping.
	Err error
}

// Error implements the error interface. The underlying cause, when present,
// is appended to the message so that it shows up in logs.
func (e *Error) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// Unwrap returns the underlying cause so that errors.Is and errors.As can
// inspect the error chain.
func (e *Error) Unwrap() error {
	return e.Err
}

// Validation returns a CodeValidation error with the given message.
func Validation(msg string) *Error {
	return &Error{Code: CodeValidation, Message: msg}
}

// Unauthorized returns a CodeUnauthorized error with the given message.
func Unauthorized(msg string) *Error {
	return &Error{Code: CodeUnauthorized, Message: msg}
}

// Forbidden returns a CodeForbidden error with the given message.
func Forbidden(msg string) *Error {
	return &Error{Code: CodeForbidden, Message: msg}
}

// NotFound returns a CodeNotFound error with the given message.
func NotFound(msg string) *Error {
	return &Error{Code: CodeNotFound, Message: msg}
}

// Conflict returns a CodeConflict error with the given message.
func Conflict(msg string) *Error {
	return &Error{Code: CodeConflict, Message: msg}
}

// RateLimited returns a CodeRateLimited error with the given message.
func RateLimited(msg string) *Error {
	return &Error{Code: CodeRateLimited, Message: msg}
}

// Internal wraps err into a CodeInternal error. The exposed message is always
// a generic one; err is only kept for logging.
func Internal(err error) *Error {
	return &Error{Code: CodeInternal, Message: internalMessage, Err: err}
}

// HTTPStatus returns the HTTP status code associated with code. Unknown codes
// map to 500 Internal Server Error.
func HTTPStatus(code Code) int {
	switch code {
	case CodeValidation:
		return nethttp.StatusBadRequest
	case CodeUnauthorized:
		return nethttp.StatusUnauthorized
	case CodeForbidden:
		return nethttp.StatusForbidden
	case CodeNotFound:
		return nethttp.StatusNotFound
	case CodeConflict:
		return nethttp.StatusConflict
	case CodeRateLimited:
		return nethttp.StatusTooManyRequests
	default:
		return nethttp.StatusInternalServerError
	}
}
