//go:build integration

package observation_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/observation"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/internal/testutil"
)

const missingID = "00000000-0000-0000-0000-000000000000"

func newObservationService(t *testing.T) (*observation.Service, *pgxpool.Pool) {
	t.Helper()

	pool := testutil.NewPool(t)
	return observation.NewService(postgres.NewObservationRepository(pool)), pool
}

func TestObservationService_CreateAndGet(t *testing.T) {
	svc, pool := newObservationService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)
	speciesID := testutil.QueryID(t, pool, `INSERT INTO species (category, common_name) VALUES ('fauna', 'Marmot') RETURNING id`)

	created, err := svc.Create(ctx, observation.Observation{
		UserID:    uid,
		SpeciesID: &speciesID,
		Location:  []byte("POINT(6 45)"),
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

func TestObservationService_List(t *testing.T) {
	svc, pool := newObservationService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)
	speciesID := testutil.QueryID(t, pool, `INSERT INTO species (category, common_name) VALUES ('fauna', 'Marmot') RETURNING id`)

	created, err := svc.Create(ctx, observation.Observation{
		UserID:    uid,
		SpeciesID: &speciesID,
		Location:  []byte("POINT(6 45)"),
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

func TestObservationService_Update(t *testing.T) {
	svc, pool := newObservationService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)
	speciesID := testutil.QueryID(t, pool, `INSERT INTO species (category, common_name) VALUES ('fauna', 'Marmot') RETURNING id`)

	e, err := svc.Create(ctx, observation.Observation{
		UserID:    uid,
		SpeciesID: &speciesID,
		Location:  []byte("POINT(6 45)"),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	e.ValidationStatus = "validated"

	got, err := svc.Update(ctx, e)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if got.ValidationStatus != "validated" {
		t.Fatalf("Update() = %+v, want updated fields", got)
	}
}

func TestObservationService_Update_NotFound(t *testing.T) {
	svc, pool := newObservationService(t)
	uid := testutil.CreateUser(t, pool)
	speciesID := testutil.QueryID(t, pool, `INSERT INTO species (category, common_name) VALUES ('fauna', 'Marmot') RETURNING id`)

	e := observation.Observation{
		UserID:    uid,
		SpeciesID: &speciesID,
		Location:  []byte("POINT(6 45)"),
	}
	e.ID = missingID

	if _, err := svc.Update(context.Background(), e); !errors.Is(err, observation.ErrNotFound) {
		t.Fatalf("Update() error = %v, want %v", err, observation.ErrNotFound)
	}
}

func TestObservationService_Delete(t *testing.T) {
	svc, pool := newObservationService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)
	speciesID := testutil.QueryID(t, pool, `INSERT INTO species (category, common_name) VALUES ('fauna', 'Marmot') RETURNING id`)

	created, err := svc.Create(ctx, observation.Observation{
		UserID:    uid,
		SpeciesID: &speciesID,
		Location:  []byte("POINT(6 45)"),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := svc.Get(ctx, created.ID); !errors.Is(err, observation.ErrNotFound) {
		t.Fatalf("Get() after Delete() error = %v, want %v", err, observation.ErrNotFound)
	}
	if err := svc.Delete(ctx, created.ID); !errors.Is(err, observation.ErrNotFound) {
		t.Fatalf("second Delete() error = %v, want %v", err, observation.ErrNotFound)
	}
}

func TestObservationService_Get_NotFound(t *testing.T) {
	svc, _ := newObservationService(t)

	if _, err := svc.Get(context.Background(), missingID); !errors.Is(err, observation.ErrNotFound) {
		t.Fatalf("Get() error = %v, want %v", err, observation.ErrNotFound)
	}
}
