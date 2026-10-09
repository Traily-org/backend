package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/user"
)

const uniqueViolationCode = "23505"

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (user.User, error) {
	row := r.pool.QueryRow(ctx, getUserQuery, id)

	u, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return user.User{}, user.ErrNotFound
	}
	if err != nil {
		return user.User{}, err
	}

	return u, nil
}

func (r *UserRepository) Create(ctx context.Context, u user.User) (user.User, error) {
	row := r.pool.QueryRow(ctx, createUserQuery, u.Email, u.Password, u.Name)

	created, err := scanUser(row)
	if isUniqueViolation(err) {
		return user.User{}, user.ErrEmailTaken
	}
	if err != nil {
		return user.User{}, err
	}

	return created, nil
}

func (r *UserRepository) Update(ctx context.Context, u user.User) (user.User, error) {
	row := r.pool.QueryRow(ctx, updateUserQuery, u.ID, u.Email, u.Password, u.Name)

	updated, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return user.User{}, user.ErrNotFound
	}
	if isUniqueViolation(err) {
		return user.User{}, user.ErrEmailTaken
	}
	if err != nil {
		return user.User{}, err
	}

	return updated, nil
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, deleteUserQuery, id)
	if err != nil {
		return err
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

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode
}
