//go:build integration

package waypoint_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/waypoint"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/internal/testutil"
)

const missingID = "00000000-0000-0000-0000-000000000000"

func newWaypointService(t *testing.T) (*waypoint.Service, *pgxpool.Pool) {
	t.Helper()

	pool := testutil.NewPool(t)
	return waypoint.NewService(postgres.NewWaypointRepository(pool)), pool
}

func TestWaypointService_CreateAndGet(t *testing.T) {
	svc, pool := newWaypointService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)
	trailID := testutil.QueryID(t, pool, `INSERT INTO trail (user_id, name, distance_m, elevation_gain_m, elevation_loss_m, estimated_duration_s, path) VALUES ($1, 'T', 1, 1, 1, 1, ST_GeomFromText('LINESTRING(0 0, 1 1)', 4326)) RETURNING id`, uid)

	created, err := svc.Create(ctx, waypoint.Waypoint{
		TrailID:    trailID,
		Label:      "Summit",
		OrderIndex: 1,
		Location:   []byte("POINT(0 0)"),
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

func TestWaypointService_List(t *testing.T) {
	svc, pool := newWaypointService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)
	trailID := testutil.QueryID(t, pool, `INSERT INTO trail (user_id, name, distance_m, elevation_gain_m, elevation_loss_m, estimated_duration_s, path) VALUES ($1, 'T', 1, 1, 1, 1, ST_GeomFromText('LINESTRING(0 0, 1 1)', 4326)) RETURNING id`, uid)

	created, err := svc.Create(ctx, waypoint.Waypoint{
		TrailID:    trailID,
		Label:      "Summit",
		OrderIndex: 1,
		Location:   []byte("POINT(0 0)"),
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

func TestWaypointService_Update(t *testing.T) {
	svc, pool := newWaypointService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)
	trailID := testutil.QueryID(t, pool, `INSERT INTO trail (user_id, name, distance_m, elevation_gain_m, elevation_loss_m, estimated_duration_s, path) VALUES ($1, 'T', 1, 1, 1, 1, ST_GeomFromText('LINESTRING(0 0, 1 1)', 4326)) RETURNING id`, uid)

	e, err := svc.Create(ctx, waypoint.Waypoint{
		TrailID:    trailID,
		Label:      "Summit",
		OrderIndex: 1,
		Location:   []byte("POINT(0 0)"),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	e.Label = "Refuge"
	e.OrderIndex = 2

	got, err := svc.Update(ctx, e)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if got.Label != "Refuge" || got.OrderIndex != 2 {
		t.Fatalf("Update() = %+v, want updated fields", got)
	}
}

func TestWaypointService_Update_NotFound(t *testing.T) {
	svc, pool := newWaypointService(t)
	uid := testutil.CreateUser(t, pool)
	trailID := testutil.QueryID(t, pool, `INSERT INTO trail (user_id, name, distance_m, elevation_gain_m, elevation_loss_m, estimated_duration_s, path) VALUES ($1, 'T', 1, 1, 1, 1, ST_GeomFromText('LINESTRING(0 0, 1 1)', 4326)) RETURNING id`, uid)

	e := waypoint.Waypoint{
		TrailID:    trailID,
		Label:      "Summit",
		OrderIndex: 1,
		Location:   []byte("POINT(0 0)"),
	}
	e.ID = missingID

	if _, err := svc.Update(context.Background(), e); !errors.Is(err, waypoint.ErrNotFound) {
		t.Fatalf("Update() error = %v, want %v", err, waypoint.ErrNotFound)
	}
}

func TestWaypointService_Delete(t *testing.T) {
	svc, pool := newWaypointService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)
	trailID := testutil.QueryID(t, pool, `INSERT INTO trail (user_id, name, distance_m, elevation_gain_m, elevation_loss_m, estimated_duration_s, path) VALUES ($1, 'T', 1, 1, 1, 1, ST_GeomFromText('LINESTRING(0 0, 1 1)', 4326)) RETURNING id`, uid)

	created, err := svc.Create(ctx, waypoint.Waypoint{
		TrailID:    trailID,
		Label:      "Summit",
		OrderIndex: 1,
		Location:   []byte("POINT(0 0)"),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := svc.Get(ctx, created.ID); !errors.Is(err, waypoint.ErrNotFound) {
		t.Fatalf("Get() after Delete() error = %v, want %v", err, waypoint.ErrNotFound)
	}
	if err := svc.Delete(ctx, created.ID); !errors.Is(err, waypoint.ErrNotFound) {
		t.Fatalf("second Delete() error = %v, want %v", err, waypoint.ErrNotFound)
	}
}

func TestWaypointService_Get_NotFound(t *testing.T) {
	svc, _ := newWaypointService(t)

	if _, err := svc.Get(context.Background(), missingID); !errors.Is(err, waypoint.ErrNotFound) {
		t.Fatalf("Get() error = %v, want %v", err, waypoint.ErrNotFound)
	}
}
