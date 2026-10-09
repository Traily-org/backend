package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/appuser"
)

type AppUserRepository struct {
	pool *pgxpool.Pool
}

func NewAppUserRepository(pool *pgxpool.Pool) *AppUserRepository {
	return &AppUserRepository{pool: pool}
}

func (r *AppUserRepository) GetByID(ctx context.Context, id string) (appuser.AppUser, error) {
	row := r.pool.QueryRow(ctx, getAppUserQuery, id)
	u, err := scanAppUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return appuser.AppUser{}, appuser.ErrNotFound
	}
	return u, err
}

func (r *AppUserRepository) GetByUsername(ctx context.Context, username string) (appuser.AppUser, error) {
	row := r.pool.QueryRow(ctx, "SELECT id, username, email, display_name, avatar_url, created_at, updated_at FROM app_user WHERE username = $1", username)
	u, err := scanAppUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return appuser.AppUser{}, appuser.ErrNotFound
	}
	return u, err
}

func (r *AppUserRepository) List(ctx context.Context) ([]appuser.AppUser, error) {
	rows, err := r.pool.Query(ctx, listAppUsersQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []appuser.AppUser
	for rows.Next() {
		u, err := scanAppUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	return users, rows.Err()
}

func (r *AppUserRepository) Create(ctx context.Context, u appuser.AppUser) (appuser.AppUser, error) {
	row := r.pool.QueryRow(ctx, createAppUserQuery, u.Username, u.Email, u.DisplayName, u.AvatarURL)
	created, err := scanAppUser(row)
	if isUniqueViolation(err) {
		return appuser.AppUser{}, appuser.ErrUsernameTaken
	}
	return created, err
}

func (r *AppUserRepository) Update(ctx context.Context, u appuser.AppUser) (appuser.AppUser, error) {
	row := r.pool.QueryRow(ctx, updateAppUserQuery, u.ID, u.Username, u.Email, u.DisplayName, u.AvatarURL)
	updated, err := scanAppUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return appuser.AppUser{}, appuser.ErrNotFound
	}
	if isUniqueViolation(err) {
		return appuser.AppUser{}, appuser.ErrUsernameTaken
	}
	return updated, err
}

func (r *AppUserRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, deleteAppUserQuery, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return appuser.ErrNotFound
	}
	return nil
}

func scanAppUser(row pgx.Row) (appuser.AppUser, error) {
	var u appuser.AppUser
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.DisplayName, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}

func isUniqueViolation(err error) bool {
	return err != nil && err.Error() != ""
}

const (
	getAppUserQuery    = "SELECT id, username, email, display_name, avatar_url, created_at, updated_at FROM app_user WHERE id = $1"
	listAppUsersQuery  = "SELECT id, username, email, display_name, avatar_url, created_at, updated_at FROM app_user ORDER BY created_at DESC"
	createAppUserQuery = "INSERT INTO app_user (username, email, display_name, avatar_url) VALUES ($1, $2, $3, $4) RETURNING id, username, email, display_name, avatar_url, created_at, updated_at"
	updateAppUserQuery = "UPDATE app_user SET username = $2, email = $3, display_name = $4, avatar_url = $5 WHERE id = $1 RETURNING id, username, email, display_name, avatar_url, created_at, updated_at"
	deleteAppUserQuery = "DELETE FROM app_user WHERE id = $1"
)
