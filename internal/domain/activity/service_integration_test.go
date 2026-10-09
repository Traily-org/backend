//go:build integration

package activity_test

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/traily-org/server/internal/domain/activity"
	"github.com/traily-org/server/internal/domain/track"
	"github.com/traily-org/server/internal/domain/user"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/migrations"
)

func newTestActivityService(t *testing.T) (
	*activity.Service,
	*track.Service,
	*user.Service,
	func(),
) {
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

	activityRepo := postgres.NewActivityRepository(pool)
	activitySvc := activity.NewService(activityRepo)

	return activitySvc, trackSvc, userSvc, cleanup
}

func TestActivityService_CreateAndGet(t *testing.T) {
	activitySvc, trackSvc, userSvc, cleanup := newTestActivityService(t)
	defer cleanup()
	ctx := context.Background()

	// Create user
	usr, err := userSvc.Create(ctx, user.User{
		Email:    "alice@example.com",
		Password: "supersecret",
		Name:     "Alice",
	})
	if err != nil {
		t.Fatalf("Create user error = %v", err)
	}

	// Create track
	distance := 100.5
	trk, err := trackSvc.Create(ctx, track.Track{
		UserID:    usr.ID,
		Route:     []byte("LINESTRING(0 0, 1 1)"),
		DistanceM: &distance,
	})
	if err != nil {
		t.Fatalf("Create track error = %v", err)
	}

	// Create activity
	created, err := activitySvc.Create(ctx, activity.Activity{
		UserID:  usr.ID,
		TrackID: trk.ID,
		Name:    ptrStr("Morning hike"),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Get activity
	fetched, err := activitySvc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if fetched.ID != created.ID || fetched.UserID != usr.ID {
		t.Fatalf("Get() returned mismatched activity")
	}
}

func TestActivityService_List(t *testing.T) {
	activitySvc, trackSvc, userSvc, cleanup := newTestActivityService(t)
	defer cleanup()
	ctx := context.Background()

	usr, _ := userSvc.Create(ctx, user.User{
		Email:    "bob@example.com",
		Password: "secret",
		Name:     "Bob",
	})
	for i := 0; i < 2; i++ {
		trk, err := trackSvc.Create(ctx, track.Track{
			UserID: usr.ID,
			Route:  []byte("LINESTRING(0 0, 1 1)"),
		})
		if err != nil {
			t.Fatalf("Create track error = %v", err)
		}
		if _, err := activitySvc.Create(ctx, activity.Activity{UserID: usr.ID, TrackID: trk.ID}); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	activities, err := activitySvc.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(activities) < 2 {
		t.Fatalf("List() returned %d activities, want at least 2", len(activities))
	}
}

func TestActivityService_Delete(t *testing.T) {
	activitySvc, trackSvc, userSvc, cleanup := newTestActivityService(t)
	defer cleanup()
	ctx := context.Background()

	usr, _ := userSvc.Create(ctx, user.User{
		Email:    "charlie@example.com",
		Password: "secret",
		Name:     "Charlie",
	})
	trk, _ := trackSvc.Create(ctx, track.Track{
		UserID: usr.ID,
		Route:  []byte("LINESTRING(0 0, 1 1)"),
	})

	created, _ := activitySvc.Create(ctx, activity.Activity{
		UserID:  usr.ID,
		TrackID: trk.ID,
	})

	if err := activitySvc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err := activitySvc.Get(ctx, created.ID)
	if err == nil {
		t.Fatalf("Get() after Delete should fail")
	}
}

func ptrStr(s string) *string {
	return &s
}
