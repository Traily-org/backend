//go:build integration

package trace_test

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/traily-org/server/internal/domain/trace"
	"github.com/traily-org/server/internal/domain/track"
	"github.com/traily-org/server/internal/domain/user"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/migrations"
)

func newTestTraceService(t *testing.T) (
	*trace.Service,
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

	traceRepo := postgres.NewTraceRepository(pool)
	traceSvc := trace.NewService(traceRepo)

	return traceSvc, trackSvc, userSvc, cleanup
}

func TestTraceService_CreateAndGet(t *testing.T) {
	traceSvc, trackSvc, userSvc, cleanup := newTestTraceService(t)
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

	trk, err := trackSvc.Create(ctx, track.Track{
		UserID: usr.ID,
		Route:  []byte(`{"type":"LineString","coordinates":[[0,0],[1,1]]}`),
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

	fetched, err := traceSvc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if fetched.Name != "Alpine Trail" {
		t.Fatalf("Get() returned mismatched trace")
	}
}

func TestTraceService_Update(t *testing.T) {
	traceSvc, trackSvc, userSvc, cleanup := newTestTraceService(t)
	defer cleanup()
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
	traceSvc, trackSvc, userSvc, cleanup := newTestTraceService(t)
	defer cleanup()
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
	traceSvc, trackSvc, userSvc, cleanup := newTestTraceService(t)
	defer cleanup()
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
