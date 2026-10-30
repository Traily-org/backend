package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/user"
)

// usersEmailConstraint is the unique constraint on users.email.
const usersEmailConstraint = "users_email_key"

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
	row := r.pool.QueryRow(ctx, getUserQuery, id)

	u, err := scanUser(row)
	if err != nil {
		return user.User{}, mapError(err, user.ErrNotFound)
	}

	return u, nil
}

// Create inserts u and returns the stored user, or user.ErrEmailTaken when the
// email is already used.
func (r *UserRepository) Create(ctx context.Context, u user.User) (user.User, error) {
	row := r.pool.QueryRow(ctx, createUserQuery, u.Email, u.Password, u.Name)

	created, err := scanUser(row)
	if isUniqueViolation(err, usersEmailConstraint) {
		return user.User{}, user.ErrEmailTaken
	}
	if err != nil {
		return user.User{}, mapError(err, user.ErrNotFound)
	}

	return created, nil
}

// Update replaces the stored fields of u and returns the updated user. It
// returns user.ErrNotFound when no user matches u.ID and user.ErrEmailTaken
// when the new email is already used.
func (r *UserRepository) Update(ctx context.Context, u user.User) (user.User, error) {
	row := r.pool.QueryRow(ctx, updateUserQuery, u.ID, u.Email, u.Password, u.Name)

	updated, err := scanUser(row)
	if isUniqueViolation(err, usersEmailConstraint) {
		return user.User{}, user.ErrEmailTaken
	}
	if err != nil {
		return user.User{}, mapError(err, user.ErrNotFound)
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

func scanUser(row pgx.Row) (user.User, error) {
	var u user.User
	err := row.Scan(&u.ID, &u.Email, &u.Password, &u.Name, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}
