//go:build integration

package publication_test

import (
	"context"
	"testing"

	"github.com/traily-org/server/internal/domain/publication"
	"github.com/traily-org/server/internal/domain/track"
	"github.com/traily-org/server/internal/domain/user"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/internal/testutil"
)

func newTestPublicationService(t *testing.T) (
	*publication.Service,
	*track.Service,
	*user.Service,
) {
	t.Helper()

	pool := testutil.NewPool(t)

	userRepo := postgres.NewUserRepository(pool)
	userSvc := user.NewService(userRepo)

	trackRepo := postgres.NewTrackRepository(pool)
	trackSvc := track.NewService(trackRepo)

	pubRepo := postgres.NewPublicationRepository(pool)
	pubSvc := publication.NewService(pubRepo)

	return pubSvc, trackSvc, userSvc
}

func TestPublicationService_CreateAndGet(t *testing.T) {
	pubSvc, trackSvc, userSvc := newTestPublicationService(t)
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
	pubSvc, trackSvc, userSvc := newTestPublicationService(t)
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
	pubSvc, trackSvc, userSvc := newTestPublicationService(t)
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
