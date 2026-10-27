//go:build integration

package photo_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/traily-org/server/internal/domain/photo"
	"github.com/traily-org/server/internal/domain/user"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/migrations"
)

func newTestServices(t *testing.T) (*photo.Service, *user.Service) {
	t.Helper()

	ctx := context.Background()

	container, err := tcpostgres.Run(ctx, "postgis/postgis:16-3.4-alpine",
		tcpostgres.WithDatabase("traily"),
		tcpostgres.WithUsername("traily"),
		tcpostgres.WithPassword("traily"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate postgres container: %v", err)
		}
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	pool, err := postgres.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := migrations.Run(ctx, pool); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	return photo.NewService(postgres.NewPhotoRepository(pool)), user.NewService(postgres.NewUserRepository(pool))
}

func TestService_CRUD(t *testing.T) {
	svc, userSvc := newTestServices(t)
	ctx := context.Background()

	usr, err := userSvc.Create(ctx, user.User{Email: "alice@example.com", Password: "supersecret", Name: "Alice"})
	if err != nil {
		t.Fatalf("user Create() error = %v", err)
	}

	created, err := svc.Create(ctx, photo.Photo{
		UserID:     usr.ID,
		StorageKey: "photos/a.jpg",
		URL:        "https://cdn.example.com/photos/a.jpg",
		Location:   []byte("POINT(6.86 45.83)"),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.NSFWStatus != "pending" || created.Visibility != "public" || string(created.Location) != "POINT(6.86 45.83)" {
		t.Fatalf("Create() = %+v", created)
	}

	noLoc, err := svc.Create(ctx, photo.Photo{UserID: usr.ID, StorageKey: "photos/b.jpg", URL: "https://cdn.example.com/photos/b.jpg"})
	if err != nil {
		t.Fatalf("Create() without location error = %v", err)
	}
	if noLoc.Location != nil {
		t.Fatalf("Location = %q, want nil", noLoc.Location)
	}

	list, err := svc.List(ctx)
	if err != nil || len(list) != 2 {
		t.Fatalf("List() = %d photos, err = %v; want 2", len(list), err)
	}

	created.NSFWStatus = "safe"
	updated, err := svc.Update(ctx, created)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.NSFWStatus != "safe" {
		t.Fatalf("NSFWStatus = %q, want safe", updated.NSFWStatus)
	}

	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := svc.Get(ctx, created.ID); !errors.Is(err, photo.ErrNotFound) {
		t.Fatalf("Get() after Delete error = %v, want ErrNotFound", err)
	}
}
