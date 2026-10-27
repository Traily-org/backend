//go:build integration

package observation_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/traily-org/server/internal/domain/observation"
	"github.com/traily-org/server/internal/domain/photo"
	"github.com/traily-org/server/internal/domain/species"
	"github.com/traily-org/server/internal/domain/user"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/migrations"
)

type services struct {
	observation *observation.Service
	photo       *photo.Service
	species     *species.Service
	user        *user.Service
}

func newTestServices(t *testing.T) services {
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

	return services{
		observation: observation.NewService(postgres.NewObservationRepository(pool)),
		photo:       photo.NewService(postgres.NewPhotoRepository(pool)),
		species:     species.NewService(postgres.NewSpeciesRepository(pool)),
		user:        user.NewService(postgres.NewUserRepository(pool)),
	}
}

func TestService_CRUD(t *testing.T) {
	s := newTestServices(t)
	ctx := context.Background()

	usr, err := s.user.Create(ctx, user.User{Email: "alice@example.com", Password: "supersecret", Name: "Alice"})
	if err != nil {
		t.Fatalf("user Create() error = %v", err)
	}
	sp, err := s.species.Create(ctx, species.Species{Category: species.CategoryPlant, CommonName: "Edelweiss"})
	if err != nil {
		t.Fatalf("species Create() error = %v", err)
	}
	ph, err := s.photo.Create(ctx, photo.Photo{UserID: usr.ID, StorageKey: "k", URL: "https://cdn.example.com/k.jpg"})
	if err != nil {
		t.Fatalf("photo Create() error = %v", err)
	}

	confidence := 0.87
	created, err := s.observation.Create(ctx, observation.Observation{
		UserID:           usr.ID,
		SpeciesID:        &sp.ID,
		PhotoID:          &ph.ID,
		Location:         []byte("POINT(6.86 45.83)"),
		ConfidenceScore:  &confidence,
		ValidationStatus: "validated",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ValidationStatus != "pending" {
		t.Fatalf("ValidationStatus = %q, want pending", created.ValidationStatus)
	}

	fetched, err := s.observation.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if *fetched.SpeciesID != sp.ID || *fetched.PhotoID != ph.ID || string(fetched.Location) != "POINT(6.86 45.83)" {
		t.Fatalf("Get() = %+v", fetched)
	}

	fetched.ValidationStatus = "validated"
	updated, err := s.observation.Update(ctx, fetched)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.ValidationStatus != "validated" {
		t.Fatalf("ValidationStatus = %q, want validated", updated.ValidationStatus)
	}

	// Supprimer la photo ne supprime pas l'observation (ON DELETE SET NULL).
	if err := s.photo.Delete(ctx, ph.ID); err != nil {
		t.Fatalf("photo Delete() error = %v", err)
	}
	after, err := s.observation.Get(ctx, created.ID)
	if err != nil || after.PhotoID != nil {
		t.Fatalf("after photo delete: PhotoID = %v, err = %v; want nil", after.PhotoID, err)
	}

	list, err := s.observation.List(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("List() = %d, err = %v; want 1", len(list), err)
	}

	if err := s.observation.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := s.observation.Get(ctx, created.ID); !errors.Is(err, observation.ErrNotFound) {
		t.Fatalf("Get() after Delete error = %v, want ErrNotFound", err)
	}
}
