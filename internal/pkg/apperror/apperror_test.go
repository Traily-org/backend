package apperror_test

import (
	"errors"
	"fmt"
	nethttp "net/http"
	"testing"

	"github.com/traily-org/server/internal/pkg/apperror"
)

func TestConstructors_ValidMessage_SetsCodeAndMessage(t *testing.T) {
	tests := []struct {
		name string
		err  *apperror.Error
		code apperror.Code
	}{
		{"validation", apperror.Validation("msg"), apperror.CodeValidation},
		{"unauthorized", apperror.Unauthorized("msg"), apperror.CodeUnauthorized},
		{"forbidden", apperror.Forbidden("msg"), apperror.CodeForbidden},
		{"not found", apperror.NotFound("msg"), apperror.CodeNotFound},
		{"conflict", apperror.Conflict("msg"), apperror.CodeConflict},
		{"rate limited", apperror.RateLimited("msg"), apperror.CodeRateLimited},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.code {
				t.Errorf("Code = %q, want %q", tt.err.Code, tt.code)
			}
			if tt.err.Message != "msg" {
				t.Errorf("Message = %q, want %q", tt.err.Message, "msg")
			}
			if tt.err.Err != nil {
				t.Errorf("Err = %v, want nil", tt.err.Err)
			}
			if tt.err.Error() != "msg" {
				t.Errorf("Error() = %q, want %q", tt.err.Error(), "msg")
			}
			if tt.err.Unwrap() != nil {
				t.Errorf("Unwrap() = %v, want nil", tt.err.Unwrap())
			}
		})
	}
}

func TestInternal_WithCause_HidesCauseFromMessage(t *testing.T) {
	cause := errors.New("connection refused")

	err := apperror.Internal(cause)

	if err.Code != apperror.CodeInternal {
		t.Errorf("Code = %q, want %q", err.Code, apperror.CodeInternal)
	}
	if err.Message != "internal server error" {
		t.Errorf("Message = %q, want generic message", err.Message)
	}
	if !errors.Is(err, cause) {
		t.Error("errors.Is(err, cause) = false, want true")
	}
	if got, want := err.Error(), "internal server error: connection refused"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestError_WrappedSentinel_MatchesByIdentity(t *testing.T) {
	sentinel := apperror.NotFound("thing not found")
	other := apperror.NotFound("thing not found")
	wrapped := fmt.Errorf("lookup: %w", sentinel)

	if !errors.Is(wrapped, sentinel) {
		t.Error("errors.Is(wrapped, sentinel) = false, want true")
	}
	if errors.Is(wrapped, other) {
		t.Error("errors.Is(wrapped, other) = true, want false (identity comparison)")
	}

	var appErr *apperror.Error
	if !errors.As(wrapped, &appErr) || appErr != sentinel {
		t.Error("errors.As did not extract the sentinel")
	}
}

func TestHTTPStatus_EveryCode_ReturnsMappedStatus(t *testing.T) {
	tests := []struct {
		code apperror.Code
		want int
	}{
		{apperror.CodeValidation, nethttp.StatusBadRequest},
		{apperror.CodeUnauthorized, nethttp.StatusUnauthorized},
		{apperror.CodeForbidden, nethttp.StatusForbidden},
		{apperror.CodeNotFound, nethttp.StatusNotFound},
		{apperror.CodeConflict, nethttp.StatusConflict},
		{apperror.CodeRateLimited, nethttp.StatusTooManyRequests},
		{apperror.CodeInternal, nethttp.StatusInternalServerError},
		{apperror.Code("SOMETHING_ELSE"), nethttp.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			if got := apperror.HTTPStatus(tt.code); got != tt.want {
				t.Errorf("HTTPStatus(%q) = %d, want %d", tt.code, got, tt.want)
			}
		})
	}
}
