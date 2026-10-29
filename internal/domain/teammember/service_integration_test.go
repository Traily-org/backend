//go:build integration

package teammember_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/teammember"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/internal/testutil"
)

const missingID = "00000000-0000-0000-0000-000000000000"

func newTeamMemberService(t *testing.T) (*teammember.Service, *pgxpool.Pool) {
	t.Helper()

	pool := testutil.NewPool(t)
	return teammember.NewService(postgres.NewTeamMemberRepository(pool)), pool
}

func TestTeamMemberService_CreateAndGet(t *testing.T) {
	svc, pool := newTeamMemberService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)
	teamID := testutil.QueryID(t, pool, `INSERT INTO team (name) VALUES ($1) RETURNING id`, testutil.Unique("team"))

	created, err := svc.Create(ctx, teammember.TeamMember{
		TeamID: teamID,
		UserID: uid,
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

func TestTeamMemberService_List(t *testing.T) {
	svc, pool := newTeamMemberService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)
	teamID := testutil.QueryID(t, pool, `INSERT INTO team (name) VALUES ($1) RETURNING id`, testutil.Unique("team"))

	created, err := svc.Create(ctx, teammember.TeamMember{
		TeamID: teamID,
		UserID: uid,
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

func TestTeamMemberService_Update(t *testing.T) {
	svc, pool := newTeamMemberService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)
	teamID := testutil.QueryID(t, pool, `INSERT INTO team (name) VALUES ($1) RETURNING id`, testutil.Unique("team"))

	e, err := svc.Create(ctx, teammember.TeamMember{
		TeamID: teamID,
		UserID: uid,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	e.Role = "admin"

	got, err := svc.Update(ctx, e)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if got.Role != "admin" {
		t.Fatalf("Update() = %+v, want updated fields", got)
	}
}

func TestTeamMemberService_Update_NotFound(t *testing.T) {
	svc, pool := newTeamMemberService(t)
	uid := testutil.CreateUser(t, pool)
	teamID := testutil.QueryID(t, pool, `INSERT INTO team (name) VALUES ($1) RETURNING id`, testutil.Unique("team"))

	e := teammember.TeamMember{
		TeamID: teamID,
		UserID: uid,
	}
	e.ID = missingID

	if _, err := svc.Update(context.Background(), e); !errors.Is(err, teammember.ErrNotFound) {
		t.Fatalf("Update() error = %v, want %v", err, teammember.ErrNotFound)
	}
}

func TestTeamMemberService_Delete(t *testing.T) {
	svc, pool := newTeamMemberService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)
	teamID := testutil.QueryID(t, pool, `INSERT INTO team (name) VALUES ($1) RETURNING id`, testutil.Unique("team"))

	created, err := svc.Create(ctx, teammember.TeamMember{
		TeamID: teamID,
		UserID: uid,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := svc.Get(ctx, created.ID); !errors.Is(err, teammember.ErrNotFound) {
		t.Fatalf("Get() after Delete() error = %v, want %v", err, teammember.ErrNotFound)
	}
	if err := svc.Delete(ctx, created.ID); !errors.Is(err, teammember.ErrNotFound) {
		t.Fatalf("second Delete() error = %v, want %v", err, teammember.ErrNotFound)
	}
}

func TestTeamMemberService_Get_NotFound(t *testing.T) {
	svc, _ := newTeamMemberService(t)

	if _, err := svc.Get(context.Background(), missingID); !errors.Is(err, teammember.ErrNotFound) {
		t.Fatalf("Get() error = %v, want %v", err, teammember.ErrNotFound)
	}
}
