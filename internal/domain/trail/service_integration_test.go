//go:build integration

package trail_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/trail"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/internal/pkg/apperror"
	"github.com/traily-org/server/internal/testutil"
)

const missingID = "00000000-0000-0000-0000-000000000000"

func newTestService(t *testing.T) (*trail.Service, *pgxpool.Pool) {
	t.Helper()

	pool := testutil.NewPool(t)
	return trail.NewService(postgres.NewTrailRepository(pool)), pool
}

func newTrail(userID, name string) trail.Trail {
	return trail.Trail{
		UserID:              &userID,
		Name:                name,
		DistanceMeters:      4200.5,
		ElevationGainMeters: 120,
		ElevationLossMeters: 118,
		Path: []trail.Coordinate{
			{Longitude: 6.129412345, Latitude: 45.899212345},
			{Longitude: 6.1301, Latitude: 45.9004},
			{Longitude: 6.1312, Latitude: 45.9011},
		},
		IsPublished: true,
	}
}

func TestService_CreateAndGet(t *testing.T) {
	svc, pool := newTestService(t)
	ctx := context.Background()
	userID := testutil.CreateUser(t, pool)

	description := "Boucle autour du lac"
	duration := 5400
	in := newTrail(userID, "Tour du lac")
	in.Description = &description
	in.Difficulty = trail.DifficultyEasy
	in.EstimatedDurationSeconds = &duration

	created, err := svc.Create(ctx, in)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID == "" {
		t.Fatal("Create() should return a generated ID")
	}
	if created.CreatedAt.IsZero() || created.UpdatedAt.IsZero() {
		t.Fatalf("Create() dates = %v / %v, want them set by the database", created.CreatedAt, created.UpdatedAt)
	}

	fetched, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if fetched.UserID == nil || *fetched.UserID != userID {
		t.Fatalf("user = %v, want %q", fetched.UserID, userID)
	}
	if fetched.Name != "Tour du lac" || fetched.Difficulty != trail.DifficultyEasy {
		t.Fatalf("Get() = %+v", fetched)
	}
	if fetched.Description == nil || *fetched.Description != description {
		t.Fatalf("description = %v, want %q", fetched.Description, description)
	}
	if fetched.EstimatedDurationSeconds == nil || *fetched.EstimatedDurationSeconds != duration {
		t.Fatalf("duration = %v, want %d", fetched.EstimatedDurationSeconds, duration)
	}
	if fetched.DistanceMeters != 4200.5 || fetched.ElevationGainMeters != 120 || fetched.ElevationLossMeters != 118 {
		t.Fatalf("metrics = %v / %v / %v", fetched.DistanceMeters, fetched.ElevationGainMeters, fetched.ElevationLossMeters)
	}
	if !fetched.IsPublished || fetched.IsOfficial {
		t.Fatalf("published = %v, official = %v", fetched.IsPublished, fetched.IsOfficial)
	}

	if len(fetched.Path) != len(in.Path) {
		t.Fatalf("path has %d points, want %d", len(fetched.Path), len(in.Path))
	}
	for i, want := range in.Path {
		if fetched.Path[i] != want {
			t.Fatalf("path[%d] = %v, want %v", i, fetched.Path[i], want)
		}
	}
}

func TestService_Create_StoresAWGS84LineString(t *testing.T) {
	svc, pool := newTestService(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, newTrail(testutil.CreateUser(t, pool), "Géométrie"))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	var geometryType string
	var srid int
	err = pool.QueryRow(ctx, `SELECT GeometryType(path), ST_SRID(path) FROM trails WHERE id = $1`, created.ID).Scan(&geometryType, &srid)
	if err != nil {
		t.Fatalf("query geometry: %v", err)
	}
	if geometryType != "LINESTRING" || srid != 4326 {
		t.Fatalf("stored geometry = %s SRID %d, want LINESTRING SRID 4326", geometryType, srid)
	}
}

func TestService_Create_Defaults(t *testing.T) {
	svc, pool := newTestService(t)
	ctx := context.Background()

	in := newTrail(testutil.CreateUser(t, pool), "Sans options")
	in.IsPublished = false
	in.IsOfficial = true

	created, err := svc.Create(ctx, in)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if created.Difficulty != trail.DifficultyModerate {
		t.Fatalf("difficulty = %q, want %q", created.Difficulty, trail.DifficultyModerate)
	}
	if created.Description != nil || created.EstimatedDurationSeconds != nil {
		t.Fatalf("optional fields = %v / %v, want nil", created.Description, created.EstimatedDurationSeconds)
	}
	if created.IsPublished {
		t.Fatal("is_published=false must be stored")
	}
	if created.IsOfficial {
		t.Fatal("a trail created through the service is never official")
	}
}

func TestService_Create_UnknownUser(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.Create(context.Background(), newTrail(missingID, "Orpheline"))

	var appErr *apperror.Error
	if !errors.As(err, &appErr) || appErr.Code != apperror.CodeValidation {
		t.Fatalf("Create() error = %v, want a validation error", err)
	}
}

func TestService_Create_InvalidTrailIsNotStored(t *testing.T) {
	svc, pool := newTestService(t)
	ctx := context.Background()

	in := newTrail(testutil.CreateUser(t, pool), "Trop courte")
	in.Path = in.Path[:1]

	if _, err := svc.Create(ctx, in); !errors.Is(err, trail.ErrPathTooShort) {
		t.Fatalf("Create() error = %v, want %v", err, trail.ErrPathTooShort)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM trails`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("stored trails = %d (err %v), want 0", count, err)
	}
}

func TestService_Get_NotFound(t *testing.T) {
	svc, _ := newTestService(t)

	if _, err := svc.Get(context.Background(), missingID); !errors.Is(err, trail.ErrNotFound) {
		t.Fatalf("Get() error = %v, want %v", err, trail.ErrNotFound)
	}
}

func TestService_Get_InvalidID(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.Get(context.Background(), "not-a-uuid")

	var appErr *apperror.Error
	if !errors.As(err, &appErr) || appErr.Code != apperror.CodeValidation {
		t.Fatalf("Get() error = %v, want a validation error", err)
	}
}

func TestService_ListByUser(t *testing.T) {
	svc, pool := newTestService(t)
	ctx := context.Background()
	alice := testutil.CreateUser(t, pool)
	bob := testutil.CreateUser(t, pool)

	var aliceIDs []string
	for _, name := range []string{"Première", "Deuxième", "Troisième"} {
		created, err := svc.Create(ctx, newTrail(alice, name))
		if err != nil {
			t.Fatalf("Create(%q) error = %v", name, err)
		}
		aliceIDs = append(aliceIDs, created.ID)
		time.Sleep(5 * time.Millisecond)
	}
	if _, err := svc.Create(ctx, newTrail(bob, "De Bob")); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	trails, err := svc.ListByUser(ctx, alice)
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}

	if len(trails) != 3 {
		t.Fatalf("ListByUser() returned %d trails, want 3", len(trails))
	}
	for i, want := range []string{aliceIDs[2], aliceIDs[1], aliceIDs[0]} {
		if trails[i].ID != want {
			t.Fatalf("trails[%d] = %s, want %s (most recent first)", i, trails[i].Name, want)
		}
		if trails[i].UserID == nil || *trails[i].UserID != alice {
			t.Fatalf("trails[%d] belongs to %v, want %q", i, trails[i].UserID, alice)
		}
		if len(trails[i].Path) != 3 {
			t.Fatalf("trails[%d] path has %d points, want 3", i, len(trails[i].Path))
		}
	}
}

func TestService_ListByUser_NoTrail(t *testing.T) {
	svc, pool := newTestService(t)

	trails, err := svc.ListByUser(context.Background(), testutil.CreateUser(t, pool))
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}
	if trails == nil || len(trails) != 0 {
		t.Fatalf("ListByUser() = %v, want an empty non-nil slice", trails)
	}
}

func TestService_TrailsSurviveTheirOwnerDeletion(t *testing.T) {
	svc, pool := newTestService(t)
	ctx := context.Background()
	userID := testutil.CreateUser(t, pool)

	created, err := svc.Create(ctx, newTrail(userID, "Orpheline plus tard"))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID); err != nil {
		t.Fatalf("delete user: %v", err)
	}

	fetched, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if fetched.UserID != nil {
		t.Fatalf("user = %v, want nil once the owner is deleted", *fetched.UserID)
	}
}
