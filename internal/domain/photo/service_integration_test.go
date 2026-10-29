//go:build integration

package photo_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/photo"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/internal/testutil"
)

const missingID = "00000000-0000-0000-0000-000000000000"

func newPhotoService(t *testing.T) (*photo.Service, *pgxpool.Pool) {
	t.Helper()

	pool := testutil.NewPool(t)
	return photo.NewService(postgres.NewPhotoRepository(pool)), pool
}

func TestPhotoService_CreateAndGet(t *testing.T) {
	svc, pool := newPhotoService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)

	created, err := svc.Create(ctx, photo.Photo{
		UserID:     uid,
		StorageKey: testutil.Unique("photos/key"),
		URL:        "https://cdn.example.com/photo.jpg",
		Location:   []byte("POINT(1 2)"),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID == "" {
		t.Fatal("Create() should return a generated ID")
	}

	got, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("Get() ID = %q, want %q", got.ID, created.ID)
	}
}

func TestPhotoService_List(t *testing.T) {
	svc, pool := newPhotoService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)

	created, err := svc.Create(ctx, photo.Photo{
		UserID:     uid,
		StorageKey: testutil.Unique("photos/key"),
		URL:        "https://cdn.example.com/photo.jpg",
		Location:   []byte("POINT(1 2)"),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	list, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	for _, item := range list {
		if item.ID == created.ID {
			return
		}
	}
	t.Fatalf("List() did not contain the created entity %q", created.ID)
}

func TestPhotoService_Update(t *testing.T) {
	svc, pool := newPhotoService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)

	e, err := svc.Create(ctx, photo.Photo{
		UserID:     uid,
		StorageKey: testutil.Unique("photos/key"),
		URL:        "https://cdn.example.com/photo.jpg",
		Location:   []byte("POINT(1 2)"),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	e.Visibility = "private"
	e.NSFWStatus = "safe"

	got, err := svc.Update(ctx, e)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if got.Visibility != "private" || got.NSFWStatus != "safe" {
		t.Fatalf("Update() = %+v, want updated fields", got)
	}
}

func TestPhotoService_Update_NotFound(t *testing.T) {
	svc, pool := newPhotoService(t)
	uid := testutil.CreateUser(t, pool)

	e := photo.Photo{
		UserID:     uid,
		StorageKey: testutil.Unique("photos/key"),
		URL:        "https://cdn.example.com/photo.jpg",
		Location:   []byte("POINT(1 2)"),
	}
	e.ID = missingID

	if _, err := svc.Update(context.Background(), e); !errors.Is(err, photo.ErrNotFound) {
		t.Fatalf("Update() error = %v, want %v", err, photo.ErrNotFound)
	}
}

func TestPhotoService_Delete(t *testing.T) {
	svc, pool := newPhotoService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)

	created, err := svc.Create(ctx, photo.Photo{
		UserID:     uid,
		StorageKey: testutil.Unique("photos/key"),
		URL:        "https://cdn.example.com/photo.jpg",
		Location:   []byte("POINT(1 2)"),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := svc.Get(ctx, created.ID); !errors.Is(err, photo.ErrNotFound) {
		t.Fatalf("Get() after Delete() error = %v, want %v", err, photo.ErrNotFound)
	}
	if err := svc.Delete(ctx, created.ID); !errors.Is(err, photo.ErrNotFound) {
		t.Fatalf("second Delete() error = %v, want %v", err, photo.ErrNotFound)
	}
}

func TestPhotoService_Get_NotFound(t *testing.T) {
	svc, _ := newPhotoService(t)

	if _, err := svc.Get(context.Background(), missingID); !errors.Is(err, photo.ErrNotFound) {
		t.Fatalf("Get() error = %v, want %v", err, photo.ErrNotFound)
	}
}
