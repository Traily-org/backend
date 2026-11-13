//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/ban"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/internal/testutil"
)

func timePtr(t time.Time) *time.Time { return &t }

// createReport inserts a report about the target and returns its ID.
func createReport(t *testing.T, pool *pgxpool.Pool, reporterID, targetType, targetID string) string {
	t.Helper()

	var id string
	err := pool.QueryRow(context.Background(), `
		INSERT INTO reports (reporter_id, target_type, target_id, reason)
		VALUES ($1, $2, $3, 'spam')
		RETURNING id::text`, reporterID, targetType, targetID).Scan(&id)
	if err != nil {
		t.Fatalf("create report: %v", err)
	}
	return id
}

func createBan(t *testing.T, pool *pgxpool.Pool, repo *postgres.BanRepository, userID, moderatorID string, expiresAt *time.Time, hash string) ban.Ban {
	t.Helper()

	reportID := createReport(t, pool, moderatorID, "user", userID)
	created, err := repo.Create(context.Background(), ban.Ban{
		UserID:    userID,
		ReportID:  &reportID,
		Reason:    "spam",
		BannedBy:  &moderatorID,
		ExpiresAt: expiresAt,
	}, []ban.Identifier{{Type: ban.IdentifierEmail, ValueHash: hash, ExpiresAt: expiresAt}})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	return created
}

func countModerationLogs(t *testing.T, pool *pgxpool.Pool, action string) int {
	t.Helper()

	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM moderation_logs WHERE action = $1`, action).Scan(&n); err != nil {
		t.Fatalf("count moderation_logs: %v", err)
	}
	return n
}

func TestBanRepository_Create_StoresBanIdentifierAndLog(t *testing.T) {
	pool := testutil.NewPool(t)
	repo := postgres.NewBanRepository(pool)
	userID := testutil.CreateUser(t, pool)
	modID := testutil.CreateUser(t, pool)
	now := time.Now().UTC().Truncate(time.Microsecond)
	expires := now.Add(24 * time.Hour)

	created := createBan(t, pool, repo, userID, modID, &expires, "hash-1")

	if created.ID == "" || created.UserID != userID || created.Reason != "spam" {
		t.Errorf("Create() = %+v", created)
	}
	if created.BannedBy == nil || *created.BannedBy != modID {
		t.Errorf("BannedBy = %v, want %s", created.BannedBy, modID)
	}
	if created.ExpiresAt == nil || !created.ExpiresAt.Equal(expires) {
		t.Errorf("ExpiresAt = %v, want %v", created.ExpiresAt, expires)
	}

	active, err := repo.Active(context.Background(), userID, now)
	if err != nil || active.ID != created.ID {
		t.Fatalf("Active() = %+v, %v, want %s", active, err, created.ID)
	}
	banned, err := repo.IdentifierBanned(context.Background(), ban.IdentifierEmail, "hash-1", now)
	if err != nil || !banned {
		t.Errorf("IdentifierBanned() = %v, %v, want true", banned, err)
	}
	if n := countModerationLogs(t, pool, "ban"); n != 1 {
		t.Errorf("ban logs = %d, want 1", n)
	}

	var status string
	var reviewedBy *string
	var logReport *string
	if err := pool.QueryRow(context.Background(),
		`SELECT r.status::text, r.reviewed_by::text, (SELECT m.report_id::text FROM moderation_logs m WHERE m.action = 'ban')
		 FROM reports r WHERE r.id = $1`, *created.ReportID).Scan(&status, &reviewedBy, &logReport); err != nil {
		t.Fatalf("read report: %v", err)
	}
	if status != "reviewed" || reviewedBy == nil || *reviewedBy != modID {
		t.Errorf("report status = %q, reviewed_by = %v, want reviewed by %s", status, reviewedBy, modID)
	}
	if logReport == nil || *logReport != *created.ReportID {
		t.Errorf("moderation log report_id = %v, want %s", logReport, *created.ReportID)
	}
}

func TestBanRepository_Report_ResolvesTargetUser(t *testing.T) {
	pool := testutil.NewPool(t)
	repo := postgres.NewBanRepository(pool)
	reporter := testutil.CreateUser(t, pool)
	author := testutil.CreateUser(t, pool)

	var photoID string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO photos (user_id, storage_key) VALUES ($1, 'photos/1.jpg') RETURNING id::text`, author).Scan(&photoID); err != nil {
		t.Fatalf("create photo: %v", err)
	}

	tests := []struct {
		name       string
		targetType string
		targetID   string
		want       *string
	}{
		{"reported user", "user", author, &author},
		{"author of a reported photo", "photo", photoID, &author},
		{"deleted target", "observation", "00000000-0000-0000-0000-000000000000", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := createReport(t, pool, reporter, tt.targetType, tt.targetID)

			got, err := repo.Report(context.Background(), id)

			if err != nil {
				t.Fatalf("Report() error = %v", err)
			}
			if got.ID != id || got.Status != ban.ReportPending {
				t.Errorf("Report() = %+v, want id %s and pending", got, id)
			}
			if (got.TargetUserID == nil) != (tt.want == nil) || (got.TargetUserID != nil && *got.TargetUserID != *tt.want) {
				t.Errorf("TargetUserID = %v, want %v", got.TargetUserID, tt.want)
			}
		})
	}

	if _, err := repo.Report(context.Background(), "00000000-0000-0000-0000-000000000000"); !errors.Is(err, ban.ErrReportNotFound) {
		t.Errorf("Report(unknown) error = %v, want ErrReportNotFound", err)
	}
}

func TestBanRepository_Active_ExpiredBan_ReturnsErrNotBanned(t *testing.T) {
	pool := testutil.NewPool(t)
	repo := postgres.NewBanRepository(pool)
	userID := testutil.CreateUser(t, pool)
	modID := testutil.CreateUser(t, pool)
	expires := time.Now().Add(time.Hour)
	createBan(t, pool, repo, userID, modID, &expires, "hash-1")

	later := expires.Add(time.Second)
	if _, err := repo.Active(context.Background(), userID, later); !errors.Is(err, ban.ErrNotBanned) {
		t.Errorf("Active(after expiry) error = %v, want ErrNotBanned", err)
	}
	if banned, err := repo.IdentifierBanned(context.Background(), ban.IdentifierEmail, "hash-1", later); err != nil || banned {
		t.Errorf("IdentifierBanned(after expiry) = %v, %v, want false", banned, err)
	}
}

func TestBanRepository_Active_SeveralBans_ReturnsTheOneEndingLast(t *testing.T) {
	pool := testutil.NewPool(t)
	repo := postgres.NewBanRepository(pool)
	userID := testutil.CreateUser(t, pool)
	modID := testutil.CreateUser(t, pool)
	now := time.Now()

	createBan(t, pool, repo, userID, modID, timePtr(now.Add(time.Hour)), "hash-1")
	permanent := createBan(t, pool, repo, userID, modID, nil, "hash-1")
	createBan(t, pool, repo, userID, modID, timePtr(now.Add(48*time.Hour)), "hash-1")

	active, err := repo.Active(context.Background(), userID, now)
	if err != nil || active.ID != permanent.ID {
		t.Fatalf("Active() = %+v, %v, want the permanent ban %s", active, err, permanent.ID)
	}
	// The identifier stays permanent although the last ban is temporary.
	if banned, _ := repo.IdentifierBanned(context.Background(), ban.IdentifierEmail, "hash-1", now.Add(72*time.Hour)); !banned {
		t.Error("IdentifierBanned() = false, want true: a permanent ban must not be shortened")
	}
}

func TestBanRepository_Lift_EndsBansAndUnbansIdentifier(t *testing.T) {
	pool := testutil.NewPool(t)
	repo := postgres.NewBanRepository(pool)
	userID := testutil.CreateUser(t, pool)
	modID := testutil.CreateUser(t, pool)
	now := time.Now()
	createBan(t, pool, repo, userID, modID, nil, "hash-1")

	n, err := repo.Lift(context.Background(), userID, modID, now)
	if err != nil || n != 1 {
		t.Fatalf("Lift() = %d, %v, want 1", n, err)
	}
	if _, err := repo.Active(context.Background(), userID, now); !errors.Is(err, ban.ErrNotBanned) {
		t.Errorf("Active() after Lift error = %v, want ErrNotBanned", err)
	}
	if banned, _ := repo.IdentifierBanned(context.Background(), ban.IdentifierEmail, "hash-1", now); banned {
		t.Error("IdentifierBanned() after Lift = true, want false")
	}
	if n := countModerationLogs(t, pool, "unban"); n != 1 {
		t.Errorf("unban logs = %d, want 1", n)
	}

	n, err = repo.Lift(context.Background(), userID, modID, now)
	if err != nil || n != 0 {
		t.Errorf("second Lift() = %d, %v, want 0", n, err)
	}
	if n := countModerationLogs(t, pool, "unban"); n != 1 {
		t.Errorf("unban logs after a no-op Lift = %d, want 1", n)
	}
}

func TestBanRepository_UserDeleted_IdentifierStaysBanned(t *testing.T) {
	pool := testutil.NewPool(t)
	repo := postgres.NewBanRepository(pool)
	userID := testutil.CreateUser(t, pool)
	modID := testutil.CreateUser(t, pool)
	createBan(t, pool, repo, userID, modID, nil, "hash-1")

	if _, err := pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID); err != nil {
		t.Fatalf("delete user: %v", err)
	}

	banned, err := repo.IdentifierBanned(context.Background(), ban.IdentifierEmail, "hash-1", time.Now())
	if err != nil || !banned {
		t.Errorf("IdentifierBanned() after account deletion = %v, %v, want true", banned, err)
	}
}
