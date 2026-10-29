//go:build integration

package plantspecies_test

import (
	"context"
	"testing"

	"github.com/traily-org/server/internal/domain/plantspecies"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/internal/testutil"
)

func newTestPlantSpeciesService(t *testing.T) *plantspecies.Service {
	t.Helper()

	pool := testutil.NewPool(t)

	psRepo := postgres.NewPlantSpeciesRepository(pool)
	psSvc := plantspecies.NewService(psRepo)

	return psSvc
}

func TestPlantSpeciesService_CreateAndGet(t *testing.T) {
	psSvc := newTestPlantSpeciesService(t)
	ctx := context.Background()

	created, err := psSvc.Create(ctx, plantspecies.PlantSpecies{
		CommonName:     "Dandelion",
		ScientificName: ptrStr("Taraxacum officinale"),
		Description:    ptrStr("Common wild flower"),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	fetched, err := psSvc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if fetched.CommonName != "Dandelion" {
		t.Fatalf("Get() returned mismatched species")
	}
}

func TestPlantSpeciesService_List(t *testing.T) {
	psSvc := newTestPlantSpeciesService(t)
	ctx := context.Background()

	psSvc.Create(ctx, plantspecies.PlantSpecies{CommonName: "Rose"})
	psSvc.Create(ctx, plantspecies.PlantSpecies{CommonName: "Tulip"})

	species, err := psSvc.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(species) < 2 {
		t.Fatalf("List() returned %d species, want at least 2", len(species))
	}
}

func TestPlantSpeciesService_Delete(t *testing.T) {
	psSvc := newTestPlantSpeciesService(t)
	ctx := context.Background()

	created, _ := psSvc.Create(ctx, plantspecies.PlantSpecies{
		CommonName: "Ivy",
	})

	if err := psSvc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err := psSvc.Get(ctx, created.ID)
	if err == nil {
		t.Fatalf("Get() after Delete should fail")
	}
}

func ptrStr(s string) *string {
	return &s
}
