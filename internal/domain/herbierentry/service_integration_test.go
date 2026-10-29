//go:build integration

package herbierentry_test

import (
	"context"
	"testing"

	"github.com/traily-org/server/internal/domain/herbier"
	"github.com/traily-org/server/internal/domain/herbierentry"
	"github.com/traily-org/server/internal/domain/plantspecies"
	"github.com/traily-org/server/internal/domain/user"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/internal/testutil"
)

func newTestHerbierEntryService(t *testing.T) (
	*herbierentry.Service,
	*herbier.Service,
	*plantspecies.Service,
	*user.Service,
) {
	t.Helper()

	pool := testutil.NewPool(t)

	userRepo := postgres.NewUserRepository(pool)
	userSvc := user.NewService(userRepo)

	herbierRepo := postgres.NewHerbierRepository(pool)
	herbierSvc := herbier.NewService(herbierRepo)

	psRepo := postgres.NewPlantSpeciesRepository(pool)
	psSvc := plantspecies.NewService(psRepo)

	heRepo := postgres.NewHerbierEntryRepository(pool)
	heSvc := herbierentry.NewService(heRepo)

	return heSvc, herbierSvc, psSvc, userSvc
}

func TestHerbierEntryService_CreateAndGet(t *testing.T) {
	heSvc, herbierSvc, psSvc, userSvc := newTestHerbierEntryService(t)
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
		HerbierID:      herb.ID,
		PlantSpeciesID: ps.ID,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	fetched, err := heSvc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if fetched.HerbierID != herb.ID {
		t.Fatalf("Get() returned mismatched entry")
	}
}

func TestHerbierEntryService_List(t *testing.T) {
	heSvc, herbierSvc, psSvc, userSvc := newTestHerbierEntryService(t)
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
		HerbierID:      herb.ID,
		PlantSpeciesID: ps1.ID,
	})
	heSvc.Create(ctx, herbierentry.HerbierEntry{
		HerbierID:      herb.ID,
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
	heSvc, herbierSvc, psSvc, userSvc := newTestHerbierEntryService(t)
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
		HerbierID:      herb.ID,
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
