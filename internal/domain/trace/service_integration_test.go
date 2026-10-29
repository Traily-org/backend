//go:build integration

package trace_test

import (
	"context"
	"testing"

	"github.com/traily-org/server/internal/domain/trace"
	"github.com/traily-org/server/internal/domain/track"
	"github.com/traily-org/server/internal/domain/user"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/internal/testutil"
)

func newTestTraceService(t *testing.T) (
	*trace.Service,
	*track.Service,
	*user.Service,
) {
	t.Helper()

	pool := testutil.NewPool(t)

	userRepo := postgres.NewUserRepository(pool)
	userSvc := user.NewService(userRepo)

	trackRepo := postgres.NewTrackRepository(pool)
	trackSvc := track.NewService(trackRepo)

	traceRepo := postgres.NewTraceRepository(pool)
	traceSvc := trace.NewService(traceRepo)

	return traceSvc, trackSvc, userSvc
}

func TestTraceService_CreateAndGet(t *testing.T) {
	traceSvc, trackSvc, userSvc := newTestTraceService(t)
	ctx := context.Background()

	usr, err := userSvc.Create(ctx, user.User{
		Email:    "alice@example.com",
		Password: "supersecret",
		Name:     "Alice",
	})
	if err != nil {
		t.Fatalf("Create user error = %v", err)
	}

	trk, err := trackSvc.Create(ctx, track.Track{
		UserID: usr.ID,
		Route:  []byte("LINESTRING(0 0, 1 1)"),
	})
	if err != nil {
		t.Fatalf("Create track error = %v", err)
	}

	created, err := traceSvc.Create(ctx, trace.Trace{
		UserID:  usr.ID,
		TrackID: trk.ID,
		Name:    "Alpine Trail",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID == "" {
		t.Fatalf("Create() returned empty ID")
	}

	fetched, err := traceSvc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if fetched.Name != "Alpine Trail" {
		t.Fatalf("Get() returned mismatched trace")
	}
}

func TestTraceService_Update(t *testing.T) {
	traceSvc, trackSvc, userSvc := newTestTraceService(t)
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

	created, _ := traceSvc.Create(ctx, trace.Trace{
		UserID:  usr.ID,
		TrackID: trk.ID,
		Name:    "Original Name",
	})

	created.Name = "Updated Name"
	updated, err := traceSvc.Update(ctx, created)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if updated.Name != "Updated Name" {
		t.Fatalf("Update() failed, got %s", updated.Name)
	}
}

func TestTraceService_List(t *testing.T) {
	traceSvc, trackSvc, userSvc := newTestTraceService(t)
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

	traceSvc.Create(ctx, trace.Trace{UserID: usr.ID, TrackID: trk.ID, Name: "Trail 1"})
	traceSvc.Create(ctx, trace.Trace{UserID: usr.ID, TrackID: trk.ID, Name: "Trail 2"})

	traces, err := traceSvc.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(traces) < 2 {
		t.Fatalf("List() returned %d traces, want at least 2", len(traces))
	}
}

func TestTraceService_Delete(t *testing.T) {
	traceSvc, trackSvc, userSvc := newTestTraceService(t)
	ctx := context.Background()

	usr, _ := userSvc.Create(ctx, user.User{
		Email:    "david@example.com",
		Password: "secret",
		Name:     "David",
	})
	trk, _ := trackSvc.Create(ctx, track.Track{
		UserID: usr.ID,
		Route:  []byte(`{"type":"LineString","coordinates":[[0,0],[1,1]]}`),
	})

	created, _ := traceSvc.Create(ctx, trace.Trace{
		UserID:  usr.ID,
		TrackID: trk.ID,
		Name:    "To Delete",
	})

	if err := traceSvc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err := traceSvc.Get(ctx, created.ID)
	if err == nil {
		t.Fatalf("Get() after Delete should fail")
	}
}
