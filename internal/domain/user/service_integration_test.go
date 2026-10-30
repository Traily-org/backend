//go:build integration

package user_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/user"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/internal/testutil"
)

func newTestService(t *testing.T) *user.Service {
	t.Helper()

	pool := testutil.NewPool(t)
	repo := postgres.NewUserRepository(pool)
	return user.NewService(repo)
}

func TestService_CreateAndGet(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	created, err := service.Create(ctx, user.User{
		Email:    "alice@example.com",
		Password: "supersecret",
		Name:     "Alice",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID == "" {
		t.Fatal("Create() should return a generated ID")
	}

	fetched, err := service.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if fetched.Email != "alice@example.com" || fetched.Name != "Alice" {
		t.Fatalf("Get() = %+v, want matching fields", fetched)
	}
}

func TestService_Create_DuplicateEmail(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	u := user.User{Email: "bob@example.com", Password: "supersecret", Name: "Bob"}

	if _, err := service.Create(ctx, u); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}

	if _, err := service.Create(ctx, u); err != user.ErrEmailTaken {
		t.Fatalf("second Create() error = %v, want %v", err, user.ErrEmailTaken)
	}
}

func TestService_Update(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	created, err := service.Create(ctx, user.User{
		Email:    "carol@example.com",
		Password: "supersecret",
		Name:     "Carol",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	created.Name = "Carol Updated"
	updated, err := service.Update(ctx, created)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Name != "Carol Updated" {
		t.Fatalf("Update() Name = %q, want %q", updated.Name, "Carol Updated")
	}
}

func TestService_Update_NotFound(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	_, err := service.Update(ctx, user.User{ID: "00000000-0000-0000-0000-000000000000", Email: "nobody@example.com", Password: "x", Name: "Nobody"})
	if err != user.ErrNotFound {
		t.Fatalf("Update() error = %v, want %v", err, user.ErrNotFound)
	}
}

func TestService_Delete(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	created, err := service.Create(ctx, user.User{
		Email:    "dave@example.com",
		Password: "supersecret",
		Name:     "Dave",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := service.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if _, err := service.Get(ctx, created.ID); err != user.ErrNotFound {
		t.Fatalf("Get() after Delete() error = %v, want %v", err, user.ErrNotFound)
	}
}

func TestService_Delete_NotFound(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	err := service.Delete(ctx, "00000000-0000-0000-0000-000000000000")
	if err != user.ErrNotFound {
		t.Fatalf("Delete() error = %v, want %v", err, user.ErrNotFound)
	}
}

func TestService_Create_StoresBcryptHash(t *testing.T) {
	pool := testutil.NewPool(t)
	service := user.NewService(postgres.NewUserRepository(pool))
	ctx := context.Background()

	created, err := service.Create(ctx, user.User{Email: "erin@example.com", Password: "supersecret", Name: "Erin"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if stored := storedPassword(t, pool, created.ID); !user.CheckPassword(stored, "supersecret") {
		t.Fatalf("stored password %q is not a bcrypt hash of the plain password", stored)
	}
}

func TestService_Update_EmptyPassword_KeepsStoredHash(t *testing.T) {
	pool := testutil.NewPool(t)
	service := user.NewService(postgres.NewUserRepository(pool))
	ctx := context.Background()

	created, err := service.Create(ctx, user.User{Email: "frank@example.com", Password: "supersecret", Name: "Frank"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	before := storedPassword(t, pool, created.ID)

	created.Name = "Frank Updated"
	if _, err := service.Update(ctx, created); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if after := storedPassword(t, pool, created.ID); after != before {
		t.Fatalf("stored password changed on update without password: %q -> %q", before, after)
	}
}

func storedPassword(t *testing.T, pool *pgxpool.Pool, id string) string {
	t.Helper()

	var password string
	if err := pool.QueryRow(context.Background(), "SELECT password FROM users WHERE id = $1", id).Scan(&password); err != nil {
		t.Fatalf("read stored password: %v", err)
	}
	return password
}
