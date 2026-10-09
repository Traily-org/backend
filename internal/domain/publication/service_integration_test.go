//go:build integration

package publication_test

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/traily-org/server/internal/domain/publication"
	"github.com/traily-org/server/internal/domain/track"
	"github.com/traily-org/server/internal/domain/user"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/migrations"
)

func newTestPublicationService(t *testing.T) (
	*publication.Service,
	*track.Service,
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

	trackRepo := postgres.NewTrackRepository(pool)
	trackSvc := track.NewService(trackRepo)

	pubRepo := postgres.NewPublicationRepository(pool)
	pubSvc := publication.NewService(pubRepo)

	return pubSvc, trackSvc, userSvc, cleanup
}

func TestPublicationService_CreateAndGet(t *testing.T) {
	pubSvc, trackSvc, userSvc, cleanup := newTestPublicationService(t)
	defer cleanup()
	ctx := context.Background()

	usr, _ := userSvc.Create(ctx, user.User{
		Email:    "alice@example.com",
		Password: "supersecret",
		Name:     "Alice",
	})
	trk, _ := trackSvc.Create(ctx, track.Track{
		UserID: usr.ID,
		Route:  []byte("LINESTRING(0 0, 1 1)"),
	})

	created, err := pubSvc.Create(ctx, publication.Publication{
		UserID:  usr.ID,
		TrackID: trk.ID,
		Title:   ptrStr("Amazing Trail"),
		Content: ptrStr("It was a great hike!"),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	fetched, err := pubSvc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if fetched.ID != created.ID {
		t.Fatalf("Get() returned mismatched publication")
	}
}

func TestPublicationService_List(t *testing.T) {
	pubSvc, trackSvc, userSvc, cleanup := newTestPublicationService(t)
	defer cleanup()
	ctx := context.Background()

	usr, _ := userSvc.Create(ctx, user.User{
		Email:    "bob@example.com",
		Password: "secret",
		Name:     "Bob",
	})
	trk, _ := trackSvc.Create(ctx, track.Track{
		UserID: usr.ID,
		Route:  []byte("LINESTRING(0 0, 1 1)"),
	})

	pubSvc.Create(ctx, publication.Publication{UserID: usr.ID, TrackID: trk.ID})
	pubSvc.Create(ctx, publication.Publication{UserID: usr.ID, TrackID: trk.ID})

	pubs, err := pubSvc.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(pubs) < 2 {
		t.Fatalf("List() returned %d publications, want at least 2", len(pubs))
	}
}

func TestPublicationService_Delete(t *testing.T) {
	pubSvc, trackSvc, userSvc, cleanup := newTestPublicationService(t)
	defer cleanup()
	ctx := context.Background()

	usr, _ := userSvc.Create(ctx, user.User{
		Email:    "charlie@example.com",
		Password: "secret",
		Name:     "Charlie",
	})
	trk, _ := trackSvc.Create(ctx, track.Track{
		UserID: usr.ID,
		Route:  []byte("LINESTRING(0 0, 1 1)"),
	})

	created, _ := pubSvc.Create(ctx, publication.Publication{
		UserID:  usr.ID,
		TrackID: trk.ID,
	})

	if err := pubSvc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err := pubSvc.Get(ctx, created.ID)
	if err == nil {
		t.Fatalf("Get() after Delete should fail")
	}
}

func ptrStr(s string) *string {
	return &s
}
