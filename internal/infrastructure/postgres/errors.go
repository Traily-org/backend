package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/traily-org/server/internal/pkg/apperror"
)

// PostgreSQL SQLSTATE codes translated into application errors.
const (
	codeInvalidTextRepresentation = "22P02"
	codeNotNullViolation          = "23502"
	codeForeignKeyViolation       = "23503"
	codeUniqueViolation           = "23505"
	codeCheckViolation            = "23514"
)

// mapError translates a pgx/PostgreSQL error into an application error so that
// no database detail ever leaks to API clients. pgx.ErrNoRows is mapped to
// notFound, known constraint violations to validation or conflict errors and
// anything else to an internal error wrapping the original one.
func mapError(err error, notFound error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return apperror.Internal(err)
	}

	switch pgErr.Code {
	case codeInvalidTextRepresentation:
		return apperror.Validation("invalid identifier")
	case codeForeignKeyViolation:
		return apperror.Validation("referenced resource does not exist")
	case codeUniqueViolation:
		return apperror.Conflict("resource already exists")
	case codeCheckViolation, codeNotNullViolation:
		return apperror.Validation("invalid value")
	default:
		return apperror.Internal(err)
	}
}

// isUniqueViolation reports whether err is a PostgreSQL unique violation of
// the given constraint.
func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == codeUniqueViolation && pgErr.ConstraintName == constraint
}
