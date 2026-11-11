package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/user"
)

// Unique constraints of the users table.
const (
	// usersEmailConstraint is the unique constraint on users.email.
	usersEmailConstraint = "users_email_key"
	// usersEmailLowerIndex is the case-insensitive unique index on users.email.
	usersEmailLowerIndex = "users_email_lower_key"
	// usersUsernameLowerIndex is the case-insensitive unique index on
	// users.username.
	usersUsernameLowerIndex = "users_username_lower_key"
	// usersUsernameConstraint is the unique constraint on users.username.
	usersUsernameConstraint = "users_username_key"
)

// UserRepository is the PostgreSQL implementation of user.Repository.
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository returns a UserRepository backed by the given pool.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// GetByID returns the user identified by id, or user.ErrNotFound.
func (r *UserRepository) GetByID(ctx context.Context, id string) (user.User, error) {
	u, err := scanUser(r.pool.QueryRow(ctx, getUserQuery, id))
	if err != nil {
		return user.User{}, mapError(err, user.ErrNotFound)
	}

	return u, nil
}

// GetByEmail returns the user whose email matches email, ignoring case, or
// user.ErrNotFound. The returned user contains the stored password hash.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (user.User, error) {
	u, err := scanUser(r.pool.QueryRow(ctx, getUserByEmailQuery, email))
	if err != nil {
		return user.User{}, mapError(err, user.ErrNotFound)
	}

	return u, nil
}

// Create inserts u and returns the stored user. It returns user.ErrEmailTaken
// or user.ErrUsernameTaken when the email or username is already used. The
// role and total points are left to their database defaults.
func (r *UserRepository) Create(ctx context.Context, u user.User) (user.User, error) {
	row := r.pool.QueryRow(ctx, createUserQuery,
		u.Email, u.Password, u.Name, u.Username, u.DisplayName, u.AvatarURL)

	created, err := scanUser(row)
	if err != nil {
		return user.User{}, mapWriteError(err)
	}

	return created, nil
}

// Update replaces the editable fields of u and returns the updated user. An
// empty u.Password keeps the stored password; the role and total points are
// never modified. It returns user.ErrNotFound when no user matches u.ID and
// user.ErrEmailTaken or user.ErrUsernameTaken when the new email or username
// is already used.
func (r *UserRepository) Update(ctx context.Context, u user.User) (user.User, error) {
	row := r.pool.QueryRow(ctx, updateUserQuery,
		u.ID, u.Email, u.Password, u.Name, u.Username, u.DisplayName, u.AvatarURL)

	updated, err := scanUser(row)
	if err != nil {
		return user.User{}, mapWriteError(err)
	}

	return updated, nil
}

// Delete removes the user identified by id, or returns user.ErrNotFound.
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, deleteUserQuery, id)
	if err != nil {
		return mapError(err, user.ErrNotFound)
	}
	if tag.RowsAffected() == 0 {
		return user.ErrNotFound
	}

	return nil
}

// mapWriteError translates an error returned by an insert or update of a user,
// mapping the unique violations on email and username to their sentinels.
func mapWriteError(err error) error {
	switch {
	case isUniqueViolation(err, usersEmailConstraint), isUniqueViolation(err, usersEmailLowerIndex):
		return user.ErrEmailTaken
	case isUniqueViolation(err, usersUsernameConstraint), isUniqueViolation(err, usersUsernameLowerIndex):
		return user.ErrUsernameTaken
	default:
		return mapError(err, user.ErrNotFound)
	}
}

// scanUser reads a user row selected in the column order shared by every
// user query.
func scanUser(row pgx.Row) (user.User, error) {
	var u user.User
	err := row.Scan(
		&u.ID, &u.Email, &u.Password, &u.Name,
		&u.Username, &u.DisplayName, &u.AvatarURL,
		&u.Role, &u.TotalPoints,
		&u.CreatedAt, &u.UpdatedAt,
	)
	return u, err
}
