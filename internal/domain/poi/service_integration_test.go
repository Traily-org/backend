//go:build integration

package poi_test

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/traily-org/server/internal/domain/poi"
	"github.com/traily-org/server/internal/domain/user"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/migrations"
)

func newTestPOIService(t *testing.T) (*poi.Service, *user.Service, func()) {
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

	poiRepo := postgres.NewPOIRepository(pool)
	poiSvc := poi.NewService(poiRepo)

	return poiSvc, userSvc, cleanup
}

func TestPOIService_CreateAndGet(t *testing.T) {
	poiSvc, userSvc, cleanup := newTestPOIService(t)
	defer cleanup()
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
		Location: []byte(`{"type":"Point","coordinates":[0,0]}`),
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
	poiSvc, userSvc, cleanup := newTestPOIService(t)
	defer cleanup()
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
		Location: []byte(`{"type":"Point","coordinates":[0,0]}`),
	})
	poiSvc.Create(ctx, poi.POI{
		UserID:   &usr.ID,
		Name:     "Lake",
		Type:     "lake",
		Location: []byte(`{"type":"Point","coordinates":[1,1]}`),
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
	poiSvc, userSvc, cleanup := newTestPOIService(t)
	defer cleanup()
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
		Location: []byte(`{"type":"Point","coordinates":[0,0]}`),
	})

	if err := poiSvc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err := poiSvc.Get(ctx, created.ID)
	if err == nil {
		t.Fatalf("Get() after Delete should fail")
	}
}
