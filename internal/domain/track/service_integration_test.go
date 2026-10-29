//go:build integration

package track_test

import (
	"context"
	"testing"

	"github.com/traily-org/server/internal/domain/track"
	"github.com/traily-org/server/internal/domain/user"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/internal/testutil"
)

func newTestTrackService(t *testing.T) (*track.Service, *user.Service) {
	t.Helper()

	pool := testutil.NewPool(t)

	userRepo := postgres.NewUserRepository(pool)
	userSvc := user.NewService(userRepo)

	trackRepo := postgres.NewTrackRepository(pool)
	trackSvc := track.NewService(trackRepo)

	return trackSvc, userSvc
}

func TestTrackService_CreateAndGet(t *testing.T) {
	trackSvc, userSvc := newTestTrackService(t)
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
	trackSvc, userSvc := newTestTrackService(t)
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
	trackSvc, userSvc := newTestTrackService(t)
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
