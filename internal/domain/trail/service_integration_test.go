//go:build integration

package trail_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/trail"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/internal/testutil"
)

const missingID = "00000000-0000-0000-0000-000000000000"

func newTrailService(t *testing.T) (*trail.Service, *pgxpool.Pool) {
	t.Helper()

	pool := testutil.NewPool(t)
	return trail.NewService(postgres.NewTrailRepository(pool)), pool
}

func TestTrailService_CreateAndGet(t *testing.T) {
	svc, pool := newTrailService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)

	created, err := svc.Create(ctx, trail.Trail{
		UserID:             &uid,
		Name:               "Mont Blanc loop",
		Difficulty:         "hard",
		DistanceM:          170000,
		ElevationGainM:     10000,
		ElevationLossM:     10000,
		EstimatedDurationS: 172800,
		Path:               []byte("LINESTRING(0 0, 1 1)"),
		IsPublished:        true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID == "" {
		t.Fatal("Create() should return a generated ID")
	}

	got, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("Get() ID = %q, want %q", got.ID, created.ID)
	}
}

func TestTrailService_List(t *testing.T) {
	svc, pool := newTrailService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)

	created, err := svc.Create(ctx, trail.Trail{
		UserID:             &uid,
		Name:               "Mont Blanc loop",
		Difficulty:         "hard",
		DistanceM:          170000,
		ElevationGainM:     10000,
		ElevationLossM:     10000,
		EstimatedDurationS: 172800,
		Path:               []byte("LINESTRING(0 0, 1 1)"),
		IsPublished:        true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	list, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	for _, item := range list {
		if item.ID == created.ID {
			return
		}
	}
	t.Fatalf("List() did not contain the created entity %q", created.ID)
}

func TestTrailService_Update(t *testing.T) {
	svc, pool := newTrailService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)

	e, err := svc.Create(ctx, trail.Trail{
		UserID:             &uid,
		Name:               "Mont Blanc loop",
		Difficulty:         "hard",
		DistanceM:          170000,
		ElevationGainM:     10000,
		ElevationLossM:     10000,
		EstimatedDurationS: 172800,
		Path:               []byte("LINESTRING(0 0, 1 1)"),
		IsPublished:        true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	e.Name = "Updated loop"
	e.Difficulty = "expert"

	got, err := svc.Update(ctx, e)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if got.Name != "Updated loop" || got.Difficulty != "expert" {
		t.Fatalf("Update() = %+v, want updated fields", got)
	}
}

func TestTrailService_Update_NotFound(t *testing.T) {
	svc, pool := newTrailService(t)
	uid := testutil.CreateUser(t, pool)

	e := trail.Trail{
		UserID:             &uid,
		Name:               "Mont Blanc loop",
		Difficulty:         "hard",
		DistanceM:          170000,
		ElevationGainM:     10000,
		ElevationLossM:     10000,
		EstimatedDurationS: 172800,
		Path:               []byte("LINESTRING(0 0, 1 1)"),
		IsPublished:        true,
	}
	e.ID = missingID

	if _, err := svc.Update(context.Background(), e); !errors.Is(err, trail.ErrNotFound) {
		t.Fatalf("Update() error = %v, want %v", err, trail.ErrNotFound)
	}
}

func TestTrailService_Delete(t *testing.T) {
	svc, pool := newTrailService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)

	created, err := svc.Create(ctx, trail.Trail{
		UserID:             &uid,
		Name:               "Mont Blanc loop",
		Difficulty:         "hard",
		DistanceM:          170000,
		ElevationGainM:     10000,
		ElevationLossM:     10000,
		EstimatedDurationS: 172800,
		Path:               []byte("LINESTRING(0 0, 1 1)"),
		IsPublished:        true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := svc.Get(ctx, created.ID); !errors.Is(err, trail.ErrNotFound) {
		t.Fatalf("Get() after Delete() error = %v, want %v", err, trail.ErrNotFound)
	}
	if err := svc.Delete(ctx, created.ID); !errors.Is(err, trail.ErrNotFound) {
		t.Fatalf("second Delete() error = %v, want %v", err, trail.ErrNotFound)
	}
}

func TestTrailService_Get_NotFound(t *testing.T) {
	svc, _ := newTrailService(t)

	if _, err := svc.Get(context.Background(), missingID); !errors.Is(err, trail.ErrNotFound) {
		t.Fatalf("Get() error = %v, want %v", err, trail.ErrNotFound)
	}
}
