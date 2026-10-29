//go:build integration

package userbadge_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/userbadge"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/internal/testutil"
)

const missingID = "00000000-0000-0000-0000-000000000000"

func newUserBadgeService(t *testing.T) (*userbadge.Service, *pgxpool.Pool) {
	t.Helper()

	pool := testutil.NewPool(t)
	return userbadge.NewService(postgres.NewUserBadgeRepository(pool)), pool
}

func TestUserBadgeService_CreateAndGet(t *testing.T) {
	svc, pool := newUserBadgeService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)
	badgeID := testutil.QueryID(t, pool, `INSERT INTO badge (code, name, description) VALUES ($1, 'B', 'D') RETURNING id`, testutil.Unique("badge"))

	created, err := svc.Create(ctx, userbadge.UserBadge{
		UserID:  uid,
		BadgeID: badgeID,
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

func TestUserBadgeService_List(t *testing.T) {
	svc, pool := newUserBadgeService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)
	badgeID := testutil.QueryID(t, pool, `INSERT INTO badge (code, name, description) VALUES ($1, 'B', 'D') RETURNING id`, testutil.Unique("badge"))

	created, err := svc.Create(ctx, userbadge.UserBadge{
		UserID:  uid,
		BadgeID: badgeID,
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

func TestUserBadgeService_Update(t *testing.T) {
	svc, pool := newUserBadgeService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)
	badgeID := testutil.QueryID(t, pool, `INSERT INTO badge (code, name, description) VALUES ($1, 'B', 'D') RETURNING id`, testutil.Unique("badge"))

	e, err := svc.Create(ctx, userbadge.UserBadge{
		UserID:  uid,
		BadgeID: badgeID,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	e.UserID = testutil.CreateUser(t, pool)

	got, err := svc.Update(ctx, e)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if got.UserID != e.UserID {
		t.Fatalf("Update() = %+v, want updated fields", got)
	}
}

func TestUserBadgeService_Update_NotFound(t *testing.T) {
	svc, pool := newUserBadgeService(t)
	uid := testutil.CreateUser(t, pool)
	badgeID := testutil.QueryID(t, pool, `INSERT INTO badge (code, name, description) VALUES ($1, 'B', 'D') RETURNING id`, testutil.Unique("badge"))

	e := userbadge.UserBadge{
		UserID:  uid,
		BadgeID: badgeID,
	}
	e.ID = missingID

	if _, err := svc.Update(context.Background(), e); !errors.Is(err, userbadge.ErrNotFound) {
		t.Fatalf("Update() error = %v, want %v", err, userbadge.ErrNotFound)
	}
}

func TestUserBadgeService_Delete(t *testing.T) {
	svc, pool := newUserBadgeService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)
	badgeID := testutil.QueryID(t, pool, `INSERT INTO badge (code, name, description) VALUES ($1, 'B', 'D') RETURNING id`, testutil.Unique("badge"))

	created, err := svc.Create(ctx, userbadge.UserBadge{
		UserID:  uid,
		BadgeID: badgeID,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := svc.Get(ctx, created.ID); !errors.Is(err, userbadge.ErrNotFound) {
		t.Fatalf("Get() after Delete() error = %v, want %v", err, userbadge.ErrNotFound)
	}
	if err := svc.Delete(ctx, created.ID); !errors.Is(err, userbadge.ErrNotFound) {
		t.Fatalf("second Delete() error = %v, want %v", err, userbadge.ErrNotFound)
	}
}

func TestUserBadgeService_Get_NotFound(t *testing.T) {
	svc, _ := newUserBadgeService(t)

	if _, err := svc.Get(context.Background(), missingID); !errors.Is(err, userbadge.ErrNotFound) {
		t.Fatalf("Get() error = %v, want %v", err, userbadge.ErrNotFound)
	}
}
