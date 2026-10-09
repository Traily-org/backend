//go:build integration

package track_test

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/traily-org/server/internal/domain/track"
	"github.com/traily-org/server/internal/domain/user"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/migrations"
)

func newTestTrackService(t *testing.T) (*track.Service, *user.Service, func()) {
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

	cleanup := func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate postgres container: %v", err)
		}
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	pool, err := postgres.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	if err := migrations.Run(ctx, pool); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	userRepo := postgres.NewUserRepository(pool)
	userSvc := user.NewService(userRepo)

	trackRepo := postgres.NewTrackRepository(pool)
	trackSvc := track.NewService(trackRepo)

	return trackSvc, userSvc, cleanup
}

func TestTrackService_CreateAndGet(t *testing.T) {
	trackSvc, userSvc, cleanup := newTestTrackService(t)
	defer cleanup()
	ctx := context.Background()

	usr, err := userSvc.Create(ctx, user.User{
		Email:    "alice@example.com",
		Password: "supersecret",
		Name:     "Alice",
	})
	if err != nil {
		t.Fatalf("Create user error = %v", err)
	}

	distance := 5.5
	created, err := trackSvc.Create(ctx, track.Track{
		UserID:    usr.ID,
		Route:     []byte("LINESTRING(0 0, 1 1)"),
		DistanceM: &distance,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	fetched, err := trackSvc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if fetched.DistanceM == nil || *fetched.DistanceM != 5.5 {
		t.Fatalf("Get() returned mismatched track")
	}
}

func TestTrackService_List(t *testing.T) {
	trackSvc, userSvc, cleanup := newTestTrackService(t)
	defer cleanup()
	ctx := context.Background()

	usr, _ := userSvc.Create(ctx, user.User{
		Email:    "bob@example.com",
		Password: "secret",
		Name:     "Bob",
	})

	trackSvc.Create(ctx, track.Track{UserID: usr.ID, Route: []byte("LINESTRING(0 0, 1 1)")})
	trackSvc.Create(ctx, track.Track{UserID: usr.ID, Route: []byte("LINESTRING(1 1, 2 2)")})

	tracks, err := trackSvc.ListByUser(ctx, usr.ID)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(tracks) < 2 {
		t.Fatalf("List() returned %d tracks, want at least 2", len(tracks))
	}
}

func TestTrackService_Delete(t *testing.T) {
	trackSvc, userSvc, cleanup := newTestTrackService(t)
	defer cleanup()
	ctx := context.Background()

	usr, _ := userSvc.Create(ctx, user.User{
		Email:    "charlie@example.com",
		Password: "secret",
		Name:     "Charlie",
	})

	created, _ := trackSvc.Create(ctx, track.Track{
		UserID: usr.ID,
		Route:  []byte("LINESTRING(0 0, 1 1)"),
	})

	if err := trackSvc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err := trackSvc.Get(ctx, created.ID)
	if err == nil {
		t.Fatalf("Get() after Delete should fail")
	}
}
