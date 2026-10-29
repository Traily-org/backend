//go:build integration

package team_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/team"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/internal/testutil"
)

const missingID = "00000000-0000-0000-0000-000000000000"

func newTeamService(t *testing.T) (*team.Service, *pgxpool.Pool) {
	t.Helper()

	pool := testutil.NewPool(t)
	return team.NewService(postgres.NewTeamRepository(pool)), pool
}

func TestTeamService_CreateAndGet(t *testing.T) {
	svc, pool := newTeamService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)

	created, err := svc.Create(ctx, team.Team{
		Name:      testutil.Unique("team"),
		CreatedBy: &uid,
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

func TestTeamService_List(t *testing.T) {
	svc, pool := newTeamService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)

	created, err := svc.Create(ctx, team.Team{
		Name:      testutil.Unique("team"),
		CreatedBy: &uid,
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

func TestTeamService_Update(t *testing.T) {
	svc, pool := newTeamService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)

	e, err := svc.Create(ctx, team.Team{
		Name:      testutil.Unique("team"),
		CreatedBy: &uid,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	e.Name = testutil.Unique("renamed")

	got, err := svc.Update(ctx, e)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if got.Name != e.Name {
		t.Fatalf("Update() = %+v, want updated fields", got)
	}
}

func TestTeamService_Update_NotFound(t *testing.T) {
	svc, pool := newTeamService(t)
	uid := testutil.CreateUser(t, pool)

	e := team.Team{
		Name:      testutil.Unique("team"),
		CreatedBy: &uid,
	}
	e.ID = missingID

	if _, err := svc.Update(context.Background(), e); !errors.Is(err, team.ErrNotFound) {
		t.Fatalf("Update() error = %v, want %v", err, team.ErrNotFound)
	}
}

func TestTeamService_Delete(t *testing.T) {
	svc, pool := newTeamService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)

	created, err := svc.Create(ctx, team.Team{
		Name:      testutil.Unique("team"),
		CreatedBy: &uid,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := svc.Get(ctx, created.ID); !errors.Is(err, team.ErrNotFound) {
		t.Fatalf("Get() after Delete() error = %v, want %v", err, team.ErrNotFound)
	}
	if err := svc.Delete(ctx, created.ID); !errors.Is(err, team.ErrNotFound) {
		t.Fatalf("second Delete() error = %v, want %v", err, team.ErrNotFound)
	}
}

func TestTeamService_Get_NotFound(t *testing.T) {
	svc, _ := newTeamService(t)

	if _, err := svc.Get(context.Background(), missingID); !errors.Is(err, team.ErrNotFound) {
		t.Fatalf("Get() error = %v, want %v", err, team.ErrNotFound)
	}
}
