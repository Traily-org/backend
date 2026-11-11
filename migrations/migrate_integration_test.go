//go:build integration

package migrations_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/testutil"
	"github.com/traily-org/server/migrations"
)

func countSQLFiles(t *testing.T) int {
	t.Helper()

	files, err := filepath.Glob("*.sql")
	if err != nil {
		t.Fatalf("glob migrations: %v", err)
	}
	return len(files)
}

func countAppliedMigrations(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()

	var n int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM schema_migrations`).Scan(&n); err != nil {
		t.Fatalf("count schema_migrations: %v", err)
	}
	return n
}

func tableExists(t *testing.T, pool *pgxpool.Pool, name string) bool {
	t.Helper()

	var exists bool
	if err := pool.QueryRow(context.Background(), `SELECT to_regclass($1) IS NOT NULL`, "public."+name).Scan(&exists); err != nil {
		t.Fatalf("check table %s: %v", name, err)
	}
	return exists
}

func totalPoints(t *testing.T, pool *pgxpool.Pool, userID string) int {
	t.Helper()

	var total int
	if err := pool.QueryRow(context.Background(), `SELECT total_points FROM users WHERE id = $1`, userID).Scan(&total); err != nil {
		t.Fatalf("read total_points: %v", err)
	}
	return total
}

func TestRun_EmptyDatabase_AppliesAllMigrations(t *testing.T) {
	pool := testutil.NewEmptyDatabase(t)
	ctx := context.Background()

	if err := migrations.Run(ctx, pool); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got, want := countAppliedMigrations(t, pool), countSQLFiles(t); got != want {
		t.Fatalf("applied migrations = %d, want %d", got, want)
	}

	for _, table := range []string{
		"users", "trail_sessions", "trails", "waypoints", "refresh_tokens",
		"species", "photos", "observations", "user_points", "badges",
		"user_badges", "teams", "team_members", "protected_zones", "reports",
		"moderation_logs",
	} {
		if !tableExists(t, pool, table) {
			t.Errorf("table %s should exist", table)
		}
	}
	for _, table := range []string{"app_user", "track"} {
		if tableExists(t, pool, table) {
			t.Errorf("table %s should have been removed or renamed", table)
		}
	}

	var referenced string
	err := pool.QueryRow(ctx, `
		SELECT confrelid::regclass::text
		FROM pg_constraint
		WHERE conname = 'trail_sessions_user_id_fkey'`).Scan(&referenced)
	if err != nil {
		t.Fatalf("read trail_sessions_user_id_fkey: %v", err)
	}
	if referenced != "users" {
		t.Fatalf("trail_sessions.user_id references %q, want %q", referenced, "users")
	}
}

func TestRun_AlreadyApplied_IsIdempotent(t *testing.T) {
	pool := testutil.NewEmptyDatabase(t)
	ctx := context.Background()

	if err := migrations.Run(ctx, pool); err != nil {
		t.Fatalf("first Run() error = %v", err)
	}
	before := countAppliedMigrations(t, pool)

	if err := migrations.Run(ctx, pool); err != nil {
		t.Fatalf("second Run() error = %v", err)
	}

	if after := countAppliedMigrations(t, pool); after != before {
		t.Fatalf("applied migrations after replay = %d, want %d", after, before)
	}
}

func TestUserPointsTrigger_InsertAndDelete_MaintainsTotalPoints(t *testing.T) {
	pool := testutil.NewPool(t)
	ctx := context.Background()
	userID := testutil.CreateUser(t, pool)

	var firstID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO user_points (user_id, points, reason) VALUES ($1, 10, 'trail_completed') RETURNING id::text`,
		userID,
	).Scan(&firstID); err != nil {
		t.Fatalf("insert first user_points: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO user_points (user_id, points, reason) VALUES ($1, 25, 'observation_validated')`,
		userID,
	); err != nil {
		t.Fatalf("insert second user_points: %v", err)
	}

	if got := totalPoints(t, pool, userID); got != 35 {
		t.Fatalf("total_points after inserts = %d, want 35", got)
	}

	if _, err := pool.Exec(ctx, `DELETE FROM user_points WHERE id = $1`, firstID); err != nil {
		t.Fatalf("delete user_points: %v", err)
	}

	if got := totalPoints(t, pool, userID); got != 25 {
		t.Fatalf("total_points after delete = %d, want 25", got)
	}
}

func TestUserPointsTrigger_UserDeleted_CascadesWithoutError(t *testing.T) {
	pool := testutil.NewPool(t)
	ctx := context.Background()
	userID := testutil.CreateUser(t, pool)

	if _, err := pool.Exec(ctx,
		`INSERT INTO user_points (user_id, points, reason) VALUES ($1, 10, 'trail_completed')`,
		userID,
	); err != nil {
		t.Fatalf("insert user_points: %v", err)
	}

	if _, err := pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID); err != nil {
		t.Fatalf("delete user: %v", err)
	}

	var remaining int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM user_points WHERE user_id = $1`, userID).Scan(&remaining); err != nil {
		t.Fatalf("count user_points: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("user_points remaining after user delete = %d, want 0", remaining)
	}
}

func TestUserPointsTrigger_Update_MovesPoints(t *testing.T) {
	pool := testutil.NewPool(t)
	ctx := context.Background()
	alice := testutil.CreateUser(t, pool)
	bob := testutil.CreateUser(t, pool)

	var id string
	if err := pool.QueryRow(ctx,
		`INSERT INTO user_points (user_id, points, reason) VALUES ($1, 10, 'trail_completed') RETURNING id::text`,
		alice,
	).Scan(&id); err != nil {
		t.Fatalf("insert user_points: %v", err)
	}

	if _, err := pool.Exec(ctx, `UPDATE user_points SET points = 7 WHERE id = $1`, id); err != nil {
		t.Fatalf("update points: %v", err)
	}
	if got := totalPoints(t, pool, alice); got != 7 {
		t.Fatalf("alice total_points after points update = %d, want 7", got)
	}

	if _, err := pool.Exec(ctx, `UPDATE user_points SET user_id = $2 WHERE id = $1`, id, bob); err != nil {
		t.Fatalf("update user_id: %v", err)
	}
	if got := totalPoints(t, pool, alice); got != 0 {
		t.Fatalf("alice total_points after move = %d, want 0", got)
	}
	if got := totalPoints(t, pool, bob); got != 7 {
		t.Fatalf("bob total_points after move = %d, want 7", got)
	}
}

func TestRun_LegacyAppUserData_MergesIntoUsers(t *testing.T) {
	pool := testutil.NewEmptyDatabase(t)
	ctx := context.Background()

	legacy, err := filepath.Glob("*.sql")
	if err != nil {
		t.Fatalf("glob legacy migrations: %v", err)
	}
	if _, err := pool.Exec(ctx, `CREATE TABLE schema_migrations (version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		t.Fatalf("create schema_migrations: %v", err)
	}
	for _, name := range legacy {
		if name >= "015" {
			continue
		}
		content, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if _, err := pool.Exec(ctx, string(content)); err != nil {
			t.Fatalf("apply %s: %v", name, err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, name); err != nil {
			t.Fatalf("record %s: %v", name, err)
		}
	}

	const (
		newAppUser   = "00000000-0000-0000-0000-0000000000a1"
		dupAppUser   = "00000000-0000-0000-0000-0000000000a2"
		existingUser = "00000000-0000-0000-0000-0000000000b1"
		trackOfNew   = "00000000-0000-0000-0000-0000000000c1"
		trackOfDup   = "00000000-0000-0000-0000-0000000000c2"
		lineString   = `ST_GeomFromText('LINESTRING(0 0, 1 1)', 4326)`
		insertTrack  = `INSERT INTO track (id, user_id, route) VALUES ($1, $2, ` + lineString + `)`
	)
	seed := []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO users (id, email, password, name) VALUES ($1, 'carol@example.com', 'hash', 'Carol')`, []any{existingUser}},
		{`INSERT INTO app_user (id, username, email) VALUES ($1, 'alice', 'alice@example.com')`, []any{newAppUser}},
		{`INSERT INTO app_user (id, username, email) VALUES ($1, 'carol', 'Carol@example.com')`, []any{dupAppUser}},
		{insertTrack, []any{trackOfNew, newAppUser}},
		{insertTrack, []any{trackOfDup, dupAppUser}},
	}
	for _, s := range seed {
		if _, err := pool.Exec(ctx, s.sql, s.args...); err != nil {
			t.Fatalf("seed %q: %v", s.sql, err)
		}
	}

	if err := migrations.Run(ctx, pool); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	owners := map[string]string{trackOfNew: newAppUser, trackOfDup: existingUser}
	for trackID, want := range owners {
		var got string
		if err := pool.QueryRow(ctx, `SELECT user_id::text FROM trail_sessions WHERE id = $1`, trackID).Scan(&got); err != nil {
			t.Fatalf("read owner of %s: %v", trackID, err)
		}
		if got != want {
			t.Fatalf("owner of %s = %s, want %s", trackID, got, want)
		}
	}

	var username, password string
	if err := pool.QueryRow(ctx, `SELECT username, password FROM users WHERE id = $1`, newAppUser).Scan(&username, &password); err != nil {
		t.Fatalf("read merged user: %v", err)
	}
	if username != "alice" || password != "!" {
		t.Fatalf("merged user = (%q, %q), want (\"alice\", \"!\")", username, password)
	}
	if tableExists(t, pool, "app_user") {
		t.Fatal("app_user still exists after migrations")
	}
}

func TestRun_EmptyDatabase_CreatesSpeciesAndZoneDetails(t *testing.T) {
	pool := testutil.NewPool(t)

	columns := map[string][]string{
		"species":         {"family", "ethical_advice"},
		"protected_zones": {"category", "regulations"},
	}
	for table, names := range columns {
		for _, name := range names {
			var exists bool
			if err := pool.QueryRow(context.Background(),
				`SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = $1 AND column_name = $2)`,
				table, name,
			).Scan(&exists); err != nil {
				t.Fatalf("check column %s.%s: %v", table, name, err)
			}
			if !exists {
				t.Errorf("column %s.%s is missing", table, name)
			}
		}
	}
}

// applyBefore applies, in order, every migration whose name sorts before
// limit and records it as applied, like Run would.
func applyBefore(t *testing.T, pool *pgxpool.Pool, limit string) {
	t.Helper()
	ctx := context.Background()

	names, err := filepath.Glob("*.sql")
	if err != nil {
		t.Fatalf("glob migrations: %v", err)
	}
	if _, err := pool.Exec(ctx, `CREATE TABLE schema_migrations (version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		t.Fatalf("create schema_migrations: %v", err)
	}
	for _, name := range names {
		if name >= limit {
			continue
		}
		content, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if _, err := pool.Exec(ctx, string(content)); err != nil {
			t.Fatalf("apply %s: %v", name, err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, name); err != nil {
			t.Fatalf("record %s: %v", name, err)
		}
	}
}

func TestRun_EmailsDifferingByCase_FailsWithExplicitMessage(t *testing.T) {
	pool := testutil.NewEmptyDatabase(t)
	ctx := context.Background()
	applyBefore(t, pool, "027")

	if _, err := pool.Exec(ctx, `INSERT INTO users (email, password, name) VALUES ('Ann@example.com', 'h', 'Ann'), ('ann@example.com', 'h', 'Ann 2')`); err != nil {
		t.Fatalf("seed users: %v", err)
	}

	err := migrations.Run(ctx, pool)
	if err == nil || !strings.Contains(err.Error(), "merged first: ann@example.com") {
		t.Fatalf("Run() error = %v, want the duplicate email to be reported", err)
	}
}

func TestRun_MixedCaseEmail_IsLowercased(t *testing.T) {
	pool := testutil.NewEmptyDatabase(t)
	ctx := context.Background()
	applyBefore(t, pool, "027")

	if _, err := pool.Exec(ctx, `INSERT INTO users (email, password, name) VALUES (' Bob@Example.com', 'h', 'Bob')`); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := migrations.Run(ctx, pool); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	var email string
	if err := pool.QueryRow(ctx, `SELECT email FROM users`).Scan(&email); err != nil {
		t.Fatalf("read email: %v", err)
	}
	if email != "bob@example.com" {
		t.Fatalf("email = %q, want %q", email, "bob@example.com")
	}
}
