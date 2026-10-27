//go:build integration

package species_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/traily-org/server/internal/domain/species"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/migrations"
)

func newTestService(t *testing.T) (*species.Service, *pgxpool.Pool) {
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
	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate postgres container: %v", err)
		}
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	pool, err := postgres.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := migrations.Run(ctx, pool); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	return species.NewService(postgres.NewSpeciesRepository(pool)), pool
}

func strPtr(s string) *string { return &s }

func TestService_CreateAndGet(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, species.Species{
		Category:          species.CategoryFungus,
		CommonName:        "Cèpe de Bordeaux",
		ScientificName:    strPtr("Boletus edulis"),
		Rarity:            "uncommon",
		ObservationMonths: []int32{8, 9, 10},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID == "" {
		t.Fatal("Create() should return a generated ID")
	}

	fetched, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if fetched.Category != species.CategoryFungus || fetched.Rarity != "uncommon" || fetched.ProtectionStatus != "none" {
		t.Fatalf("Get() = %+v", fetched)
	}
	if len(fetched.ObservationMonths) != 3 || fetched.ObservationMonths[0] != 8 {
		t.Fatalf("ObservationMonths = %v, want [8 9 10]", fetched.ObservationMonths)
	}
}

func TestService_ListUpdateDelete(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	ortie, _ := svc.Create(ctx, species.Species{Category: species.CategoryPlant, CommonName: "Ortie"})
	_, _ = svc.Create(ctx, species.Species{Category: species.CategoryFungus, CommonName: "Girolle"})

	list, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list) != 2 || list[0].CommonName != "Girolle" {
		t.Fatalf("List() = %v, want 2 species sorted by name", list)
	}

	ortie.Rarity = "rare"
	updated, err := svc.Update(ctx, ortie)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Rarity != "rare" || !updated.UpdatedAt.After(ortie.UpdatedAt) {
		t.Fatalf("Update() = %+v", updated)
	}

	if err := svc.Delete(ctx, ortie.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := svc.Get(ctx, ortie.ID); !errors.Is(err, species.ErrNotFound) {
		t.Fatalf("Get() after Delete error = %v, want ErrNotFound", err)
	}
	if err := svc.Delete(ctx, ortie.ID); !errors.Is(err, species.ErrNotFound) {
		t.Fatalf("second Delete() error = %v, want ErrNotFound", err)
	}
}

// Vérifie la migration 024 : catégories plant/fungus seulement, ancien modèle herbier supprimé.
func TestMigration_PlantsAndFungiOnly(t *testing.T) {
	_, pool := newTestService(t)
	ctx := context.Background()

	if _, err := postgres.NewSpeciesRepository(pool).Create(ctx, species.Species{
		Category: "fauna", CommonName: "Renard", ProtectionStatus: "none", Rarity: "common",
	}); err == nil {
		t.Fatal("the database should reject the fauna category")
	}

	for _, table := range []string{"herbier", "herbier_entry", "plant_species"} {
		var exists bool
		if err := pool.QueryRow(ctx, "SELECT to_regclass($1) IS NOT NULL", table).Scan(&exists); err != nil {
			t.Fatal(err)
		}
		if exists {
			t.Fatalf("table %s should have been dropped", table)
		}
	}
}
