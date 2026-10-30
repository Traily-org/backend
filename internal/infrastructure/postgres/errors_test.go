package postgres

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/traily-org/server/internal/pkg/apperror"
)

func TestMapError_NilError_ReturnsNil(t *testing.T) {
	if err := mapError(nil, errors.New("not found")); err != nil {
		t.Fatalf("mapError(nil) = %v, want nil", err)
	}
}

func TestMapError_NoRows_ReturnsNotFoundSentinel(t *testing.T) {
	notFound := apperror.NotFound("thing not found")

	err := mapError(fmt.Errorf("scan: %w", pgx.ErrNoRows), notFound)

	if !errors.Is(err, notFound) {
		t.Fatalf("mapError(ErrNoRows) = %v, want %v", err, notFound)
	}
}

func TestMapError_PgErrorCodes_ReturnsAppError(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		want    apperror.Code
		message string
	}{
		{"invalid text representation", "22P02", apperror.CodeValidation, "invalid identifier"},
		{"foreign key violation", "23503", apperror.CodeValidation, "referenced resource does not exist"},
		{"unique violation", "23505", apperror.CodeConflict, "resource already exists"},
		{"check violation", "23514", apperror.CodeValidation, "invalid value"},
		{"not null violation", "23502", apperror.CodeValidation, "invalid value"},
		{"unknown code", "40001", apperror.CodeInternal, "internal server error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pgErr := &pgconn.PgError{Code: tt.code, Message: "secret database detail"}

			err := mapError(fmt.Errorf("query: %w", pgErr), nil)

			var appErr *apperror.Error
			if !errors.As(err, &appErr) {
				t.Fatalf("mapError() = %T, want *apperror.Error", err)
			}
			if appErr.Code != tt.want {
				t.Errorf("Code = %q, want %q", appErr.Code, tt.want)
			}
			if appErr.Message != tt.message {
				t.Errorf("Message = %q, want %q", appErr.Message, tt.message)
			}
		})
	}
}

func TestMapError_UnknownPgError_KeepsCauseForLogging(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "53300", Message: "too many connections"}

	err := mapError(pgErr, nil)

	if !errors.Is(err, pgErr) {
		t.Fatalf("mapError() should wrap the original error, got %v", err)
	}
}

func TestMapError_NonPgError_ReturnsInternal(t *testing.T) {
	cause := errors.New("connection reset")

	err := mapError(cause, nil)

	var appErr *apperror.Error
	if !errors.As(err, &appErr) || appErr.Code != apperror.CodeInternal {
		t.Fatalf("mapError() = %v, want internal error", err)
	}
	if !errors.Is(err, cause) {
		t.Fatal("mapError() should wrap the original error")
	}
}

func TestIsUniqueViolation_VariousErrors_DetectsOnly23505OnConstraint(t *testing.T) {
	if !isUniqueViolation(fmt.Errorf("wrap: %w", &pgconn.PgError{Code: "23505", ConstraintName: "users_email_key"}), "users_email_key") {
		t.Error("isUniqueViolation(23505, users_email_key) = false, want true")
	}
	if isUniqueViolation(&pgconn.PgError{Code: "23505", ConstraintName: "users_username_key"}, "users_email_key") {
		t.Error("isUniqueViolation(23505, users_username_key) = true, want false")
	}
	if isUniqueViolation(&pgconn.PgError{Code: "23503", ConstraintName: "users_email_key"}, "users_email_key") {
		t.Error("isUniqueViolation(23503) = true, want false")
	}
	if isUniqueViolation(errors.New("boom"), "users_email_key") {
		t.Error("isUniqueViolation(plain error) = true, want false")
	}
}
