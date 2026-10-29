//go:build integration

package report_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/report"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/internal/testutil"
)

const missingID = "00000000-0000-0000-0000-000000000000"

func newReportService(t *testing.T) (*report.Service, *pgxpool.Pool) {
	t.Helper()

	pool := testutil.NewPool(t)
	return report.NewService(postgres.NewReportRepository(pool)), pool
}

func TestReportService_CreateAndGet(t *testing.T) {
	svc, pool := newReportService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)
	targetID := testutil.CreateUser(t, pool)

	created, err := svc.Create(ctx, report.Report{
		ReporterID: uid,
		TargetType: "user",
		TargetID:   targetID,
		Reason:     "spam",
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

func TestReportService_List(t *testing.T) {
	svc, pool := newReportService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)
	targetID := testutil.CreateUser(t, pool)

	created, err := svc.Create(ctx, report.Report{
		ReporterID: uid,
		TargetType: "user",
		TargetID:   targetID,
		Reason:     "spam",
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

func TestReportService_Update(t *testing.T) {
	svc, pool := newReportService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)
	targetID := testutil.CreateUser(t, pool)

	e, err := svc.Create(ctx, report.Report{
		ReporterID: uid,
		TargetType: "user",
		TargetID:   targetID,
		Reason:     "spam",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	e.Status = "reviewed"
	e.ReviewedBy = &uid

	got, err := svc.Update(ctx, e)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if got.Status != "reviewed" || got.ReviewedBy == nil {
		t.Fatalf("Update() = %+v, want updated fields", got)
	}
}

func TestReportService_Update_NotFound(t *testing.T) {
	svc, pool := newReportService(t)
	uid := testutil.CreateUser(t, pool)
	targetID := testutil.CreateUser(t, pool)

	e := report.Report{
		ReporterID: uid,
		TargetType: "user",
		TargetID:   targetID,
		Reason:     "spam",
	}
	e.ID = missingID

	if _, err := svc.Update(context.Background(), e); !errors.Is(err, report.ErrNotFound) {
		t.Fatalf("Update() error = %v, want %v", err, report.ErrNotFound)
	}
}

func TestReportService_Delete(t *testing.T) {
	svc, pool := newReportService(t)
	ctx := context.Background()
	uid := testutil.CreateUser(t, pool)
	targetID := testutil.CreateUser(t, pool)

	created, err := svc.Create(ctx, report.Report{
		ReporterID: uid,
		TargetType: "user",
		TargetID:   targetID,
		Reason:     "spam",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := svc.Get(ctx, created.ID); !errors.Is(err, report.ErrNotFound) {
		t.Fatalf("Get() after Delete() error = %v, want %v", err, report.ErrNotFound)
	}
	if err := svc.Delete(ctx, created.ID); !errors.Is(err, report.ErrNotFound) {
		t.Fatalf("second Delete() error = %v, want %v", err, report.ErrNotFound)
	}
}

func TestReportService_Get_NotFound(t *testing.T) {
	svc, _ := newReportService(t)

	if _, err := svc.Get(context.Background(), missingID); !errors.Is(err, report.ErrNotFound) {
		t.Fatalf("Get() error = %v, want %v", err, report.ErrNotFound)
	}
}
