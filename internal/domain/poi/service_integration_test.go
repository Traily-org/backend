//go:build integration

package poi_test

import (
	"context"
	"testing"

	"github.com/traily-org/server/internal/domain/poi"
	"github.com/traily-org/server/internal/domain/user"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/internal/testutil"
)

func newTestPOIService(t *testing.T) (*poi.Service, *user.Service) {
	t.Helper()

	pool := testutil.NewPool(t)

	userRepo := postgres.NewUserRepository(pool)
	userSvc := user.NewService(userRepo)

	poiRepo := postgres.NewPOIRepository(pool)
	poiSvc := poi.NewService(poiRepo)

	return poiSvc, userSvc
}

func TestPOIService_CreateAndGet(t *testing.T) {
	poiSvc, userSvc := newTestPOIService(t)
	ctx := context.Background()

	usr, err := userSvc.Create(ctx, user.User{
		Email:    "alice@example.com",
		Password: "supersecret",
		Name:     "Alice",
	})
	if err != nil {
		t.Fatalf("Create user error = %v", err)
	}

	created, err := poiSvc.Create(ctx, poi.POI{
		UserID:   &usr.ID,
		Name:     "Mountain Peak",
		Type:     "viewpoint",
		Location: []byte("POINT(0 0)"),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	fetched, err := poiSvc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if fetched.Name != "Mountain Peak" {
		t.Fatalf("Get() returned mismatched POI")
	}
}

func TestPOIService_List(t *testing.T) {
	poiSvc, userSvc := newTestPOIService(t)
	ctx := context.Background()

	usr, _ := userSvc.Create(ctx, user.User{
		Email:    "bob@example.com",
		Password: "secret",
		Name:     "Bob",
	})

	poiSvc.Create(ctx, poi.POI{
		UserID:   &usr.ID,
		Name:     "Waterfall",
		Type:     "waterfall",
		Location: []byte("POINT(0 0)"),
	})
	poiSvc.Create(ctx, poi.POI{
		UserID:   &usr.ID,
		Name:     "Lake",
		Type:     "lake",
		Location: []byte("POINT(1 1)"),
	})

	pois, err := poiSvc.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(pois) < 2 {
		t.Fatalf("List() returned %d POIs, want at least 2", len(pois))
	}
}

func TestPOIService_Delete(t *testing.T) {
	poiSvc, userSvc := newTestPOIService(t)
	ctx := context.Background()

	usr, _ := userSvc.Create(ctx, user.User{
		Email:    "charlie@example.com",
		Password: "secret",
		Name:     "Charlie",
	})

	created, _ := poiSvc.Create(ctx, poi.POI{
		UserID:   &usr.ID,
		Name:     "To Delete",
		Type:     "other",
		Location: []byte("POINT(0 0)"),
	})

	if err := poiSvc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err := poiSvc.Get(ctx, created.ID)
	if err == nil {
		t.Fatalf("Get() after Delete should fail")
	}
}
