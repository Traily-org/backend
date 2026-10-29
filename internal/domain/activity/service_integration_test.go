//go:build integration

package activity_test

import (
	"context"
	"testing"

	"github.com/traily-org/server/internal/domain/activity"
	"github.com/traily-org/server/internal/domain/track"
	"github.com/traily-org/server/internal/domain/user"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/internal/testutil"
)

func newTestActivityService(t *testing.T) (
	*activity.Service,
	*track.Service,
	*user.Service,
) {
	t.Helper()

	pool := testutil.NewPool(t)

	userRepo := postgres.NewUserRepository(pool)
	userSvc := user.NewService(userRepo)

	trackRepo := postgres.NewTrackRepository(pool)
	trackSvc := track.NewService(trackRepo)

	activityRepo := postgres.NewActivityRepository(pool)
	activitySvc := activity.NewService(activityRepo)

	return activitySvc, trackSvc, userSvc
}

func TestActivityService_CreateAndGet(t *testing.T) {
	activitySvc, trackSvc, userSvc := newTestActivityService(t)
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
	activitySvc, trackSvc, userSvc := newTestActivityService(t)
	ctx := context.Background()

	usr, _ := userSvc.Create(ctx, user.User{
		Email:    "bob@example.com",
		Password: "secret",
		Name:     "Bob",
	})
	trk, _ := trackSvc.Create(ctx, track.Track{
		UserID: usr.ID,
		Route:  []byte(`{"type":"LineString","coordinates":[[0,0],[1,1]]}`),
	})

	activitySvc.Create(ctx, activity.Activity{UserID: usr.ID, TrackID: trk.ID})
	activitySvc.Create(ctx, activity.Activity{UserID: usr.ID, TrackID: trk.ID})

	activities, err := activitySvc.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(activities) < 2 {
		t.Fatalf("List() returned %d activities, want at least 2", len(activities))
	}
}

func TestActivityService_Delete(t *testing.T) {
	activitySvc, trackSvc, userSvc := newTestActivityService(t)
	ctx := context.Background()

	usr, _ := userSvc.Create(ctx, user.User{
		Email:    "charlie@example.com",
		Password: "secret",
		Name:     "Charlie",
	})
	trk, _ := trackSvc.Create(ctx, track.Track{
		UserID: usr.ID,
		Route:  []byte(`{"type":"LineString","coordinates":[[0,0],[1,1]]}`),
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
