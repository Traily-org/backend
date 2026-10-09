//go:build integration

package plantspecies_test

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/traily-org/server/internal/domain/plantspecies"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/migrations"
)

func newTestPlantSpeciesService(t *testing.T) (*plantspecies.Service, func()) {
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

	psRepo := postgres.NewPlantSpeciesRepository(pool)
	psSvc := plantspecies.NewService(psRepo)

	return psSvc, cleanup
}

func TestPlantSpeciesService_CreateAndGet(t *testing.T) {
	psSvc, cleanup := newTestPlantSpeciesService(t)
	defer cleanup()
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
	psSvc, cleanup := newTestPlantSpeciesService(t)
	defer cleanup()
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
	psSvc, cleanup := newTestPlantSpeciesService(t)
	defer cleanup()
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
