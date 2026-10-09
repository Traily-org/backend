//go:build integration

package herbierentry_test

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/traily-org/server/internal/domain/herbier"
	"github.com/traily-org/server/internal/domain/herbierentry"
	"github.com/traily-org/server/internal/domain/plantspecies"
	"github.com/traily-org/server/internal/domain/user"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/migrations"
)

func newTestHerbierEntryService(t *testing.T) (
	*herbierentry.Service,
	*herbier.Service,
	*plantspecies.Service,
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

	herbierRepo := postgres.NewHerbierRepository(pool)
	herbierSvc := herbier.NewService(herbierRepo)

	psRepo := postgres.NewPlantSpeciesRepository(pool)
	psSvc := plantspecies.NewService(psRepo)

	heRepo := postgres.NewHerbierEntryRepository(pool)
	heSvc := herbierentry.NewService(heRepo)

	return heSvc, herbierSvc, psSvc, userSvc, cleanup
}

func TestHerbierEntryService_CreateAndGet(t *testing.T) {
	heSvc, herbierSvc, psSvc, userSvc, cleanup := newTestHerbierEntryService(t)
	defer cleanup()
	ctx := context.Background()

	usr, _ := userSvc.Create(ctx, user.User{
		Email:    "alice@example.com",
		Password: "supersecret",
		Name:     "Alice",
	})

	herb, _ := herbierSvc.Create(ctx, herbier.Herbier{
		UserID: usr.ID,
		Name:   "My Collection",
	})

	ps, _ := psSvc.Create(ctx, plantspecies.PlantSpecies{
		CommonName: "Rose",
	})

	created, err := heSvc.Create(ctx, herbierentry.HerbierEntry{
		HerbierId:      herb.ID,
		PlantSpeciesID: ps.ID,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	fetched, err := heSvc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if fetched.HerbierId != herb.ID {
		t.Fatalf("Get() returned mismatched entry")
	}
}

func TestHerbierEntryService_List(t *testing.T) {
	heSvc, herbierSvc, psSvc, userSvc, cleanup := newTestHerbierEntryService(t)
	defer cleanup()
	ctx := context.Background()

	usr, _ := userSvc.Create(ctx, user.User{
		Email:    "bob@example.com",
		Password: "secret",
		Name:     "Bob",
	})

	herb, _ := herbierSvc.Create(ctx, herbier.Herbier{
		UserID: usr.ID,
		Name:   "Collection 1",
	})

	ps1, _ := psSvc.Create(ctx, plantspecies.PlantSpecies{CommonName: "Daisy"})
	ps2, _ := psSvc.Create(ctx, plantspecies.PlantSpecies{CommonName: "Lily"})

	heSvc.Create(ctx, herbierentry.HerbierEntry{
		HerbierId:      herb.ID,
		PlantSpeciesID: ps1.ID,
	})
	heSvc.Create(ctx, herbierentry.HerbierEntry{
		HerbierId:      herb.ID,
		PlantSpeciesID: ps2.ID,
	})

	entries, err := heSvc.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(entries) < 2 {
		t.Fatalf("List() returned %d entries, want at least 2", len(entries))
	}
}

func TestHerbierEntryService_Delete(t *testing.T) {
	heSvc, herbierSvc, psSvc, userSvc, cleanup := newTestHerbierEntryService(t)
	defer cleanup()
	ctx := context.Background()

	usr, _ := userSvc.Create(ctx, user.User{
		Email:    "charlie@example.com",
		Password: "secret",
		Name:     "Charlie",
	})

	herb, _ := herbierSvc.Create(ctx, herbier.Herbier{
		UserID: usr.ID,
		Name:   "Collection 2",
	})

	ps, _ := psSvc.Create(ctx, plantspecies.PlantSpecies{
		CommonName: "Sunflower",
	})

	created, _ := heSvc.Create(ctx, herbierentry.HerbierEntry{
		HerbierId:      herb.ID,
		PlantSpeciesID: ps.ID,
	})

	if err := heSvc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err := heSvc.Get(ctx, created.ID)
	if err == nil {
		t.Fatalf("Get() after Delete should fail")
	}
}
