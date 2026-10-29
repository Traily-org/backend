//go:build integration

package herbier_test

import (
	"context"
	"testing"

	"github.com/traily-org/server/internal/domain/herbier"
	"github.com/traily-org/server/internal/domain/user"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/internal/testutil"
)

func newTestHerbierService(t *testing.T) (*herbier.Service, *user.Service) {
	t.Helper()

	pool := testutil.NewPool(t)

	userRepo := postgres.NewUserRepository(pool)
	userSvc := user.NewService(userRepo)

	herbierRepo := postgres.NewHerbierRepository(pool)
	herbierSvc := herbier.NewService(herbierRepo)

	return herbierSvc, userSvc
}

func TestHerbierService_CreateAndGet(t *testing.T) {
	herbierSvc, userSvc := newTestHerbierService(t)
	ctx := context.Background()

	usr, err := userSvc.Create(ctx, user.User{
		Email:    "alice@example.com",
		Password: "supersecret",
		Name:     "Alice",
	})
	if err != nil {
		t.Fatalf("Create user error = %v", err)
	}

	created, err := herbierSvc.Create(ctx, herbier.Herbier{
		UserID: usr.ID,
		Name:   "My Herbier",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	fetched, err := herbierSvc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if fetched.Name != "My Herbier" {
		t.Fatalf("Get() returned mismatched herbier")
	}
}

func TestHerbierService_List(t *testing.T) {
	herbierSvc, userSvc := newTestHerbierService(t)
	ctx := context.Background()

	usr, _ := userSvc.Create(ctx, user.User{
		Email:    "bob@example.com",
		Password: "secret",
		Name:     "Bob",
	})

	herbierSvc.Create(ctx, herbier.Herbier{UserID: usr.ID, Name: "Herbier 1"})

	herbiers, err := herbierSvc.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(herbiers) < 1 {
		t.Fatalf("List() returned %d herbiers, want at least 1", len(herbiers))
	}
}

func TestHerbierService_Delete(t *testing.T) {
	herbierSvc, userSvc := newTestHerbierService(t)
	ctx := context.Background()

	usr, _ := userSvc.Create(ctx, user.User{
		Email:    "charlie@example.com",
		Password: "secret",
		Name:     "Charlie",
	})

	created, _ := herbierSvc.Create(ctx, herbier.Herbier{
		UserID: usr.ID,
		Name:   "To Delete",
	})

	if err := herbierSvc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err := herbierSvc.Get(ctx, created.ID)
	if err == nil {
		t.Fatalf("Get() after Delete should fail")
	}
}
