//go:build integration

package refreshtoken_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/refreshtoken"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/internal/testutil"
)

const missingID = "00000000-0000-0000-0000-000000000000"

func newRefreshTokenService(t *testing.T) (*refreshtoken.Service, *pgxpool.Pool) {
	t.Helper()

	pool := testutil.NewPool(t)
	return refreshtoken.NewService(postgres.NewRefreshTokenRepository(pool)), pool
}

func TestRefreshTokenService_CreateAndGet(t *testing.T) {
	svc, pool := newRefreshTokenService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)

	created, err := svc.Create(ctx, refreshtoken.RefreshToken{
		UserID:    uid,
		TokenHash: testutil.Unique("hash"),
		ExpiresAt: time.Now().Add(time.Hour).UTC().Truncate(time.Second),
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

func TestRefreshTokenService_List(t *testing.T) {
	svc, pool := newRefreshTokenService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)

	created, err := svc.Create(ctx, refreshtoken.RefreshToken{
		UserID:    uid,
		TokenHash: testutil.Unique("hash"),
		ExpiresAt: time.Now().Add(time.Hour).UTC().Truncate(time.Second),
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

func TestRefreshTokenService_Update(t *testing.T) {
	svc, pool := newRefreshTokenService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)

	e, err := svc.Create(ctx, refreshtoken.RefreshToken{
		UserID:    uid,
		TokenHash: testutil.Unique("hash"),
		ExpiresAt: time.Now().Add(time.Hour).UTC().Truncate(time.Second),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	e.RevokedAt = &now

	got, err := svc.Update(ctx, e)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if got.RevokedAt == nil {
		t.Fatalf("Update() = %+v, want updated fields", got)
	}
}

func TestRefreshTokenService_Update_NotFound(t *testing.T) {
	svc, pool := newRefreshTokenService(t)
	uid := testutil.CreateUser(t, pool)

	e := refreshtoken.RefreshToken{
		UserID:    uid,
		TokenHash: testutil.Unique("hash"),
		ExpiresAt: time.Now().Add(time.Hour).UTC().Truncate(time.Second),
	}
	e.ID = missingID

	if _, err := svc.Update(context.Background(), e); !errors.Is(err, refreshtoken.ErrNotFound) {
		t.Fatalf("Update() error = %v, want %v", err, refreshtoken.ErrNotFound)
	}
}

func TestRefreshTokenService_Delete(t *testing.T) {
	svc, pool := newRefreshTokenService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)

	created, err := svc.Create(ctx, refreshtoken.RefreshToken{
		UserID:    uid,
		TokenHash: testutil.Unique("hash"),
		ExpiresAt: time.Now().Add(time.Hour).UTC().Truncate(time.Second),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := svc.Get(ctx, created.ID); !errors.Is(err, refreshtoken.ErrNotFound) {
		t.Fatalf("Get() after Delete() error = %v, want %v", err, refreshtoken.ErrNotFound)
	}
	if err := svc.Delete(ctx, created.ID); !errors.Is(err, refreshtoken.ErrNotFound) {
		t.Fatalf("second Delete() error = %v, want %v", err, refreshtoken.ErrNotFound)
	}
}

func TestRefreshTokenService_Get_NotFound(t *testing.T) {
	svc, _ := newRefreshTokenService(t)

	if _, err := svc.Get(context.Background(), missingID); !errors.Is(err, refreshtoken.ErrNotFound) {
		t.Fatalf("Get() error = %v, want %v", err, refreshtoken.ErrNotFound)
	}
}
