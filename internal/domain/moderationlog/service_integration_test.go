//go:build integration

package moderationlog_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/moderationlog"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/internal/testutil"
)

const missingID = "00000000-0000-0000-0000-000000000000"

func newModerationLogService(t *testing.T) (*moderationlog.Service, *pgxpool.Pool) {
	t.Helper()

	pool := testutil.NewPool(t)
	return moderationlog.NewService(postgres.NewModerationLogRepository(pool)), pool
}

func TestModerationLogService_CreateAndGet(t *testing.T) {
	svc, pool := newModerationLogService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)
	targetID := testutil.CreateUser(t, pool)

	created, err := svc.Create(ctx, moderationlog.ModerationLog{
		ModeratorID: uid,
		Action:      "ban",
		TargetType:  "user",
		TargetID:    targetID,
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

func TestModerationLogService_List(t *testing.T) {
	svc, pool := newModerationLogService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)
	targetID := testutil.CreateUser(t, pool)

	created, err := svc.Create(ctx, moderationlog.ModerationLog{
		ModeratorID: uid,
		Action:      "ban",
		TargetType:  "user",
		TargetID:    targetID,
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

func TestModerationLogService_Update(t *testing.T) {
	svc, pool := newModerationLogService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)
	targetID := testutil.CreateUser(t, pool)

	e, err := svc.Create(ctx, moderationlog.ModerationLog{
		ModeratorID: uid,
		Action:      "ban",
		TargetType:  "user",
		TargetID:    targetID,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	e.Action = "reject"

	got, err := svc.Update(ctx, e)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if got.Action != "reject" {
		t.Fatalf("Update() = %+v, want updated fields", got)
	}
}

func TestModerationLogService_Update_NotFound(t *testing.T) {
	svc, pool := newModerationLogService(t)
	uid := testutil.CreateUser(t, pool)
	targetID := testutil.CreateUser(t, pool)

	e := moderationlog.ModerationLog{
		ModeratorID: uid,
		Action:      "ban",
		TargetType:  "user",
		TargetID:    targetID,
	}
	e.ID = missingID

	if _, err := svc.Update(context.Background(), e); !errors.Is(err, moderationlog.ErrNotFound) {
		t.Fatalf("Update() error = %v, want %v", err, moderationlog.ErrNotFound)
	}
}

func TestModerationLogService_Delete(t *testing.T) {
	svc, pool := newModerationLogService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)
	targetID := testutil.CreateUser(t, pool)

	created, err := svc.Create(ctx, moderationlog.ModerationLog{
		ModeratorID: uid,
		Action:      "ban",
		TargetType:  "user",
		TargetID:    targetID,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := svc.Get(ctx, created.ID); !errors.Is(err, moderationlog.ErrNotFound) {
		t.Fatalf("Get() after Delete() error = %v, want %v", err, moderationlog.ErrNotFound)
	}
	if err := svc.Delete(ctx, created.ID); !errors.Is(err, moderationlog.ErrNotFound) {
		t.Fatalf("second Delete() error = %v, want %v", err, moderationlog.ErrNotFound)
	}
}

func TestModerationLogService_Get_NotFound(t *testing.T) {
	svc, _ := newModerationLogService(t)

	if _, err := svc.Get(context.Background(), missingID); !errors.Is(err, moderationlog.ErrNotFound) {
		t.Fatalf("Get() error = %v, want %v", err, moderationlog.ErrNotFound)
	}
}
