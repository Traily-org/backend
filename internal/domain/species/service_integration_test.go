//go:build integration

package species_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/species"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/internal/testutil"
)

const missingID = "00000000-0000-0000-0000-000000000000"

func newSpeciesService(t *testing.T) (*species.Service, *pgxpool.Pool) {
	t.Helper()

	pool := testutil.NewPool(t)
	return species.NewService(postgres.NewSpeciesRepository(pool)), pool
}

func TestSpeciesService_CreateAndGet(t *testing.T) {
	svc, _ := newSpeciesService(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, species.Species{
		Category:          "flora",
		CommonName:        "Edelweiss",
		ProtectionStatus:  "vulnerable",
		ObservationMonths: []int32{6, 7, 8},
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

func TestSpeciesService_List(t *testing.T) {
	svc, _ := newSpeciesService(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, species.Species{
		Category:          "flora",
		CommonName:        "Edelweiss",
		ProtectionStatus:  "vulnerable",
		ObservationMonths: []int32{6, 7, 8},
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

func TestSpeciesService_Update(t *testing.T) {
	svc, _ := newSpeciesService(t)
	ctx := context.Background()

	e, err := svc.Create(ctx, species.Species{
		Category:          "flora",
		CommonName:        "Edelweiss",
		ProtectionStatus:  "vulnerable",
		ObservationMonths: []int32{6, 7, 8},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	e.CommonName = "Alpine edelweiss"
	e.Rarity = "rare"

	got, err := svc.Update(ctx, e)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if got.CommonName != "Alpine edelweiss" || got.Rarity != "rare" {
		t.Fatalf("Update() = %+v, want updated fields", got)
	}
}

func TestSpeciesService_Update_NotFound(t *testing.T) {
	svc, _ := newSpeciesService(t)

	e := species.Species{
		Category:          "flora",
		CommonName:        "Edelweiss",
		ProtectionStatus:  "vulnerable",
		ObservationMonths: []int32{6, 7, 8},
	}
	e.ID = missingID

	if _, err := svc.Update(context.Background(), e); !errors.Is(err, species.ErrNotFound) {
		t.Fatalf("Update() error = %v, want %v", err, species.ErrNotFound)
	}
}

func TestSpeciesService_Delete(t *testing.T) {
	svc, _ := newSpeciesService(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, species.Species{
		Category:          "flora",
		CommonName:        "Edelweiss",
		ProtectionStatus:  "vulnerable",
		ObservationMonths: []int32{6, 7, 8},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := svc.Get(ctx, created.ID); !errors.Is(err, species.ErrNotFound) {
		t.Fatalf("Get() after Delete() error = %v, want %v", err, species.ErrNotFound)
	}
	if err := svc.Delete(ctx, created.ID); !errors.Is(err, species.ErrNotFound) {
		t.Fatalf("second Delete() error = %v, want %v", err, species.ErrNotFound)
	}
}

func TestSpeciesService_Get_NotFound(t *testing.T) {
	svc, _ := newSpeciesService(t)

	if _, err := svc.Get(context.Background(), missingID); !errors.Is(err, species.ErrNotFound) {
		t.Fatalf("Get() error = %v, want %v", err, species.ErrNotFound)
	}
}
