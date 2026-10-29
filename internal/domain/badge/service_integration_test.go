//go:build integration

package badge_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/badge"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/internal/testutil"
)

const missingID = "00000000-0000-0000-0000-000000000000"

func newBadgeService(t *testing.T) (*badge.Service, *pgxpool.Pool) {
	t.Helper()

	pool := testutil.NewPool(t)
	return badge.NewService(postgres.NewBadgeRepository(pool)), pool
}

func TestBadgeService_CreateAndGet(t *testing.T) {
	svc, _ := newBadgeService(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, badge.Badge{
		Code:        testutil.Unique("badge"),
		Name:        "Explorer",
		Description: "Record your first track",
		Criteria:    []byte(`{"tracks": 1}`),
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

func TestBadgeService_List(t *testing.T) {
	svc, _ := newBadgeService(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, badge.Badge{
		Code:        testutil.Unique("badge"),
		Name:        "Explorer",
		Description: "Record your first track",
		Criteria:    []byte(`{"tracks": 1}`),
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

func TestBadgeService_Update(t *testing.T) {
	svc, _ := newBadgeService(t)
	ctx := context.Background()

	e, err := svc.Create(ctx, badge.Badge{
		Code:        testutil.Unique("badge"),
		Name:        "Explorer",
		Description: "Record your first track",
		Criteria:    []byte(`{"tracks": 1}`),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	e.Name = "Seasoned explorer"

	got, err := svc.Update(ctx, e)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if got.Name != "Seasoned explorer" {
		t.Fatalf("Update() = %+v, want updated fields", got)
	}
}

func TestBadgeService_Update_NotFound(t *testing.T) {
	svc, _ := newBadgeService(t)

	e := badge.Badge{
		Code:        testutil.Unique("badge"),
		Name:        "Explorer",
		Description: "Record your first track",
		Criteria:    []byte(`{"tracks": 1}`),
	}
	e.ID = missingID

	if _, err := svc.Update(context.Background(), e); !errors.Is(err, badge.ErrNotFound) {
		t.Fatalf("Update() error = %v, want %v", err, badge.ErrNotFound)
	}
}

func TestBadgeService_Delete(t *testing.T) {
	svc, _ := newBadgeService(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, badge.Badge{
		Code:        testutil.Unique("badge"),
		Name:        "Explorer",
		Description: "Record your first track",
		Criteria:    []byte(`{"tracks": 1}`),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := svc.Get(ctx, created.ID); !errors.Is(err, badge.ErrNotFound) {
		t.Fatalf("Get() after Delete() error = %v, want %v", err, badge.ErrNotFound)
	}
	if err := svc.Delete(ctx, created.ID); !errors.Is(err, badge.ErrNotFound) {
		t.Fatalf("second Delete() error = %v, want %v", err, badge.ErrNotFound)
	}
}

func TestBadgeService_Get_NotFound(t *testing.T) {
	svc, _ := newBadgeService(t)

	if _, err := svc.Get(context.Background(), missingID); !errors.Is(err, badge.ErrNotFound) {
		t.Fatalf("Get() error = %v, want %v", err, badge.ErrNotFound)
	}
}
