//go:build integration

package protectedzone_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/protectedzone"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/internal/testutil"
)

const missingID = "00000000-0000-0000-0000-000000000000"

func newProtectedZoneService(t *testing.T) (*protectedzone.Service, *pgxpool.Pool) {
	t.Helper()

	pool := testutil.NewPool(t)
	return protectedzone.NewService(postgres.NewProtectedZoneRepository(pool)), pool
}

func TestProtectedZoneService_CreateAndGet(t *testing.T) {
	svc, _ := newProtectedZoneService(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, protectedzone.ProtectedZone{
		Name:     "Vanoise",
		Category: "national_park",
		Boundary: []byte("MULTIPOLYGON(((0 0, 0 1, 1 1, 1 0, 0 0)))"),
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

func TestProtectedZoneService_List(t *testing.T) {
	svc, _ := newProtectedZoneService(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, protectedzone.ProtectedZone{
		Name:     "Vanoise",
		Category: "national_park",
		Boundary: []byte("MULTIPOLYGON(((0 0, 0 1, 1 1, 1 0, 0 0)))"),
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

func TestProtectedZoneService_Update(t *testing.T) {
	svc, _ := newProtectedZoneService(t)
	ctx := context.Background()

	e, err := svc.Create(ctx, protectedzone.ProtectedZone{
		Name:     "Vanoise",
		Category: "national_park",
		Boundary: []byte("MULTIPOLYGON(((0 0, 0 1, 1 1, 1 0, 0 0)))"),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	e.Name = "Vanoise core"

	got, err := svc.Update(ctx, e)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if got.Name != "Vanoise core" {
		t.Fatalf("Update() = %+v, want updated fields", got)
	}
}

func TestProtectedZoneService_Update_NotFound(t *testing.T) {
	svc, _ := newProtectedZoneService(t)

	e := protectedzone.ProtectedZone{
		Name:     "Vanoise",
		Category: "national_park",
		Boundary: []byte("MULTIPOLYGON(((0 0, 0 1, 1 1, 1 0, 0 0)))"),
	}
	e.ID = missingID

	if _, err := svc.Update(context.Background(), e); !errors.Is(err, protectedzone.ErrNotFound) {
		t.Fatalf("Update() error = %v, want %v", err, protectedzone.ErrNotFound)
	}
}

func TestProtectedZoneService_Delete(t *testing.T) {
	svc, _ := newProtectedZoneService(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, protectedzone.ProtectedZone{
		Name:     "Vanoise",
		Category: "national_park",
		Boundary: []byte("MULTIPOLYGON(((0 0, 0 1, 1 1, 1 0, 0 0)))"),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := svc.Get(ctx, created.ID); !errors.Is(err, protectedzone.ErrNotFound) {
		t.Fatalf("Get() after Delete() error = %v, want %v", err, protectedzone.ErrNotFound)
	}
	if err := svc.Delete(ctx, created.ID); !errors.Is(err, protectedzone.ErrNotFound) {
		t.Fatalf("second Delete() error = %v, want %v", err, protectedzone.ErrNotFound)
	}
}

func TestProtectedZoneService_Get_NotFound(t *testing.T) {
	svc, _ := newProtectedZoneService(t)

	if _, err := svc.Get(context.Background(), missingID); !errors.Is(err, protectedzone.ErrNotFound) {
		t.Fatalf("Get() error = %v, want %v", err, protectedzone.ErrNotFound)
	}
}
