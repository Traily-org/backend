//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/auth"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/internal/testutil"
)

func createRefreshToken(t *testing.T, repo *postgres.RefreshTokenRepository, userID, hash string) auth.RefreshToken {
	t.Helper()

	created, err := repo.Create(context.Background(), auth.RefreshToken{
		UserID:    userID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour).UTC().Truncate(time.Microsecond),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	return created
}

func getRefreshToken(t *testing.T, repo *postgres.RefreshTokenRepository, hash string) auth.RefreshToken {
	t.Helper()

	got, err := repo.GetByHash(context.Background(), hash)
	if err != nil {
		t.Fatalf("GetByHash(%q) error = %v", hash, err)
	}
	return got
}

func TestRefreshTokenRepository_CreateThenGetByHash_ReturnsStoredToken(t *testing.T) {
	pool := testutil.NewPool(t)
	repo := postgres.NewRefreshTokenRepository(pool)
	userID := testutil.CreateUser(t, pool)
	expiresAt := time.Now().Add(30 * 24 * time.Hour).UTC().Truncate(time.Microsecond)

	created, err := repo.Create(context.Background(), auth.RefreshToken{
		UserID:    userID,
		TokenHash: "hash-1",
		ExpiresAt: expiresAt,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if created.ID == "" || created.CreatedAt.IsZero() {
		t.Errorf("Create() = %+v, want generated ID and CreatedAt", created)
	}
	if created.RevokedAt != nil {
		t.Errorf("Create() RevokedAt = %v, want nil", created.RevokedAt)
	}

	got := getRefreshToken(t, repo, "hash-1")
	if got.ID != created.ID || got.UserID != userID || got.TokenHash != "hash-1" {
		t.Errorf("GetByHash() = %+v, want %+v", got, created)
	}
	if !got.ExpiresAt.Equal(expiresAt) {
		t.Errorf("GetByHash() ExpiresAt = %v, want %v", got.ExpiresAt, expiresAt)
	}
}

func TestRefreshTokenRepository_GetByHashUnknown_ReturnsErrRefreshTokenNotFound(t *testing.T) {
	pool := testutil.NewPool(t)
	repo := postgres.NewRefreshTokenRepository(pool)

	_, err := repo.GetByHash(context.Background(), "unknown-hash")

	if !errors.Is(err, auth.ErrRefreshTokenNotFound) {
		t.Fatalf("GetByHash() error = %v, want ErrRefreshTokenNotFound", err)
	}
}

func TestRefreshTokenRepository_Revoke_ReturnsTrueThenFalse(t *testing.T) {
	pool := testutil.NewPool(t)
	repo := postgres.NewRefreshTokenRepository(pool)
	userID := testutil.CreateUser(t, pool)
	token := createRefreshToken(t, repo, userID, "hash-1")
	first := time.Now().UTC().Truncate(time.Microsecond)

	revoked, err := repo.Revoke(context.Background(), token.ID, first)
	if err != nil || !revoked {
		t.Fatalf("Revoke() first call = %v, %v; want true, nil", revoked, err)
	}
	revoked, err = repo.Revoke(context.Background(), token.ID, first.Add(time.Minute))
	if err != nil || revoked {
		t.Fatalf("Revoke() second call = %v, %v; want false, nil", revoked, err)
	}

	got := getRefreshToken(t, repo, "hash-1")
	if got.RevokedAt == nil || !got.RevokedAt.Equal(first) {
		t.Errorf("RevokedAt = %v, want %v (first revocation kept)", got.RevokedAt, first)
	}
}

func TestRefreshTokenRepository_RevokeAllForUser_LeavesOtherUsersUntouched(t *testing.T) {
	pool := testutil.NewPool(t)
	repo := postgres.NewRefreshTokenRepository(pool)
	alice := testutil.CreateUser(t, pool)
	bob := testutil.CreateUser(t, pool)
	createRefreshToken(t, repo, alice, "alice-1")
	createRefreshToken(t, repo, alice, "alice-2")
	createRefreshToken(t, repo, bob, "bob-1")

	if err := repo.RevokeAllForUser(context.Background(), alice, time.Now()); err != nil {
		t.Fatalf("RevokeAllForUser() error = %v", err)
	}

	for _, hash := range []string{"alice-1", "alice-2"} {
		if got := getRefreshToken(t, repo, hash); got.RevokedAt == nil {
			t.Errorf("token %s RevokedAt = nil, want revoked", hash)
		}
	}
	if got := getRefreshToken(t, repo, "bob-1"); got.RevokedAt != nil {
		t.Errorf("token bob-1 RevokedAt = %v, want nil", got.RevokedAt)
	}
}

func TestRefreshTokenRepository_UserDeleted_CascadesToTokens(t *testing.T) {
	pool := testutil.NewPool(t)
	repo := postgres.NewRefreshTokenRepository(pool)
	userID := testutil.CreateUser(t, pool)
	createRefreshToken(t, repo, userID, "hash-1")

	deleteUser(t, pool, userID)

	if _, err := repo.GetByHash(context.Background(), "hash-1"); !errors.Is(err, auth.ErrRefreshTokenNotFound) {
		t.Fatalf("GetByHash() after user deletion error = %v, want ErrRefreshTokenNotFound", err)
	}
}

func deleteUser(t *testing.T, pool *pgxpool.Pool, id string) {
	t.Helper()

	if _, err := pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, id); err != nil {
		t.Fatalf("delete user: %v", err)
	}
}
