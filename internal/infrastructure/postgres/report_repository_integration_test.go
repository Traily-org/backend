//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/report"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/internal/testutil"
)

func insertID(t *testing.T, pool *pgxpool.Pool, query string, args ...any) string {
	t.Helper()

	var id string
	if err := pool.QueryRow(context.Background(), query, args...).Scan(&id); err != nil {
		t.Fatalf("insert fixture: %v", err)
	}
	return id
}

func TestReportRepository_Create_StoresEveryTraceabilityField(t *testing.T) {
	pool := testutil.NewPool(t)
	repo := postgres.NewReportRepository(pool)
	reporter := testutil.CreateUser(t, pool)
	target := testutil.CreateUser(t, pool)

	created, err := repo.Create(context.Background(), report.Report{
		ReporterID: reporter, TargetType: report.TargetUser, TargetID: target, Reason: "harassment",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if created.ID == "" || created.CreatedAt.IsZero() {
		t.Errorf("Create() = %+v, want ID and CreatedAt set", created)
	}
	if created.ReporterID != reporter || created.TargetType != report.TargetUser || created.TargetID != target || created.Reason != "harassment" {
		t.Errorf("Create() = %+v, want reporter, target and reason stored", created)
	}
	if created.Status != report.StatusPending || created.ReviewedBy != nil || created.ReviewedAt != nil {
		t.Errorf("Create() = %+v, want pending and not reviewed", created)
	}
}

func TestReportRepository_Create_DuplicatePending_ReturnsErrAlreadyReported(t *testing.T) {
	pool := testutil.NewPool(t)
	repo := postgres.NewReportRepository(pool)
	reporter := testutil.CreateUser(t, pool)
	target := testutil.CreateUser(t, pool)
	rep := report.Report{ReporterID: reporter, TargetType: report.TargetUser, TargetID: target, Reason: "spam"}

	if _, err := repo.Create(context.Background(), rep); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}
	if _, err := repo.Create(context.Background(), rep); !errors.Is(err, report.ErrAlreadyReported) {
		t.Fatalf("second Create() error = %v, want ErrAlreadyReported", err)
	}

	// Once handled, the same user can report the target again.
	if _, err := pool.Exec(context.Background(), `UPDATE reports SET status = 'dismissed'`); err != nil {
		t.Fatalf("dismiss report: %v", err)
	}
	if _, err := repo.Create(context.Background(), rep); err != nil {
		t.Errorf("Create() after dismissal error = %v, want nil", err)
	}
}

func TestReportRepository_TargetAuthor_RespectsVisibility(t *testing.T) {
	pool := testutil.NewPool(t)
	repo := postgres.NewReportRepository(pool)
	reporter := testutil.CreateUser(t, pool)
	author := testutil.CreateUser(t, pool)

	publicPhoto := insertID(t, pool, `INSERT INTO photos (user_id, storage_key) VALUES ($1, 'photos/public.jpg') RETURNING id::text`, author)
	privatePhoto := insertID(t, pool, `INSERT INTO photos (user_id, storage_key, visibility) VALUES ($1, 'photos/private.jpg', 'private') RETURNING id::text`, author)
	friendsPhoto := insertID(t, pool, `INSERT INTO photos (user_id, storage_key, visibility) VALUES ($1, 'photos/friends.jpg', 'friends') RETURNING id::text`, author)
	observation := insertID(t, pool, `INSERT INTO observations (user_id, location) VALUES ($1, ST_SetSRID(ST_MakePoint(2.35, 48.85), 4326)) RETURNING id::text`, author)
	privateObservation := insertID(t, pool, `INSERT INTO observations (user_id, location, visibility) VALUES ($1, ST_SetSRID(ST_MakePoint(2.35, 48.85), 4326), 'private') RETURNING id::text`, author)
	trail := insertID(t, pool, trailInsert, author, true)
	draftTrail := insertID(t, pool, trailInsert, author, false)
	anonymousTrail := insertID(t, pool, `
		INSERT INTO trails (name, distance_meters, elevation_gain_meters, elevation_loss_meters, path)
		VALUES ('official', 1, 0, 0, ST_GeomFromText('LINESTRING(0 0, 1 1)', 4326))
		RETURNING id::text`)

	visible := []struct {
		kind report.TargetType
		id   string
		want *string
	}{
		{report.TargetUser, author, &author},
		{report.TargetPhoto, publicPhoto, &author},
		{report.TargetObservation, observation, &author},
		{report.TargetTrail, trail, &author},
		{report.TargetTrail, anonymousTrail, nil},
	}
	for _, tt := range visible {
		got, err := repo.TargetAuthor(context.Background(), tt.kind, tt.id, reporter)
		if err != nil {
			t.Errorf("TargetAuthor(%s) error = %v", tt.kind, err)
			continue
		}
		if (got == nil) != (tt.want == nil) || (got != nil && *got != *tt.want) {
			t.Errorf("TargetAuthor(%s) = %v, want %v", tt.kind, got, tt.want)
		}
	}

	hidden := []struct {
		kind report.TargetType
		id   string
	}{
		{report.TargetPhoto, privatePhoto},
		{report.TargetPhoto, friendsPhoto},
		{report.TargetObservation, privateObservation},
		{report.TargetTrail, draftTrail},
		{report.TargetUser, "00000000-0000-4000-8000-000000000000"},
		{report.TargetPhoto, "00000000-0000-4000-8000-000000000000"},
	}
	for _, tt := range hidden {
		if _, err := repo.TargetAuthor(context.Background(), tt.kind, tt.id, reporter); !errors.Is(err, report.ErrTargetNotFound) {
			t.Errorf("TargetAuthor(%s, %s) error = %v, want ErrTargetNotFound", tt.kind, tt.id, err)
		}
	}

	// The author still sees their own private content.
	if got, err := repo.TargetAuthor(context.Background(), report.TargetPhoto, privatePhoto, author); err != nil || got == nil || *got != author {
		t.Errorf("TargetAuthor(own private photo) = %v, %v, want the author", got, err)
	}
}

const trailInsert = `
	INSERT INTO trails (user_id, name, distance_meters, elevation_gain_meters, elevation_loss_meters, path, is_published)
	VALUES ($1, 'loop', 1, 0, 0, ST_GeomFromText('LINESTRING(0 0, 1 1)', 4326), $2)
	RETURNING id::text`

func TestReportRepository_List_FiltersOrdersAndPaginates(t *testing.T) {
	pool := testutil.NewPool(t)
	repo := postgres.NewReportRepository(pool)
	reporter := testutil.CreateUser(t, pool)

	targets := make([]string, 3)
	for i := range targets {
		targets[i] = testutil.CreateUser(t, pool)
		if _, err := repo.Create(context.Background(), report.Report{
			ReporterID: reporter, TargetType: report.TargetUser, TargetID: targets[i], Reason: "spam",
		}); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}
	photo := insertID(t, pool, `INSERT INTO photos (user_id, storage_key) VALUES ($1, 'photos/1.jpg') RETURNING id::text`, targets[0])
	if _, err := repo.Create(context.Background(), report.Report{
		ReporterID: reporter, TargetType: report.TargetPhoto, TargetID: photo, Reason: "nsfw",
	}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE reports SET status = 'dismissed' WHERE target_id = $1`, targets[0]); err != nil {
		t.Fatalf("dismiss report: %v", err)
	}

	all, err := repo.List(context.Background(), report.Filter{Limit: 10})
	if err != nil || all.Total != 4 || len(all.Items) != 4 {
		t.Fatalf("List() = %+v, %v, want 4 reports", all, err)
	}
	if all.Items[0].TargetID != photo {
		t.Errorf("first report target = %s, want the newest (%s)", all.Items[0].TargetID, photo)
	}

	pending := report.StatusPending
	got, err := repo.List(context.Background(), report.Filter{Status: &pending, Limit: 10})
	if err != nil || got.Total != 3 {
		t.Errorf("List(pending) = %+v, %v, want 3 reports", got, err)
	}

	kind := report.TargetPhoto
	got, err = repo.List(context.Background(), report.Filter{TargetType: &kind, Limit: 10})
	if err != nil || got.Total != 1 || got.Items[0].Reason != "nsfw" {
		t.Errorf("List(photo) = %+v, %v, want the photo report", got, err)
	}

	got, err = repo.List(context.Background(), report.Filter{Limit: 3, Offset: 3})
	if err != nil || got.Total != 4 || len(got.Items) != 1 {
		t.Errorf("List(page 2) = %+v, %v, want total 4 with 1 item", got, err)
	}
	got, err = repo.List(context.Background(), report.Filter{Limit: 3, Offset: 50})
	if err != nil || got.Total != 4 || len(got.Items) != 0 {
		t.Errorf("List(past the end) = %+v, %v, want total 4 with no items", got, err)
	}
}

func TestReportService_CreatedReportIsListedImmediately(t *testing.T) {
	pool := testutil.NewPool(t)
	svc := report.NewService(postgres.NewReportRepository(pool))
	reporter := testutil.CreateUser(t, pool)
	target := testutil.CreateUser(t, pool)

	if _, err := svc.Create(context.Background(), report.Input{
		ReporterID: reporter, TargetType: report.TargetUser, TargetID: target, Reason: "harassment",
	}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	pending := report.StatusPending
	page, err := svc.List(context.Background(), report.Filter{Status: &pending})
	if err != nil || page.Total != 1 {
		t.Fatalf("List() = %+v, %v, want the new report", page, err)
	}
	if got := page.Items[0]; got.ReporterID != reporter || got.TargetID != target || got.Reason != "harassment" || got.CreatedAt.IsZero() {
		t.Errorf("List() item = %+v, want reporter, target, reason and date", got)
	}

	if _, err := svc.Create(context.Background(), report.Input{
		ReporterID: reporter, TargetType: report.TargetUser, TargetID: reporter, Reason: "me",
	}); !errors.Is(err, report.ErrCannotReportSelf) {
		t.Errorf("Create(self) error = %v, want ErrCannotReportSelf", err)
	}
}
