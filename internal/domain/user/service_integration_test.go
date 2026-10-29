//go:build integration

package user_test

import (
	"context"
	"testing"

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

func TestService_Profile(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	username, displayName := "alice42", "Alice B."
	created, err := service.Create(ctx, user.User{
		Email:       "profile@example.com",
		Password:    "supersecret",
		Name:        "Alice",
		Username:    &username,
		DisplayName: &displayName,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	fetched, err := service.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if fetched.Username == nil || *fetched.Username != username || fetched.DisplayName == nil || *fetched.DisplayName != displayName {
		t.Fatalf("Get() = %+v, want username %q and display name %q", fetched, username, displayName)
	}
	if fetched.Role != "user" || fetched.TotalPoints != 0 {
		t.Fatalf("Get() role = %q, points = %d, want defaults", fetched.Role, fetched.TotalPoints)
	}
}
