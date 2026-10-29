//go:build integration

// Package testutil shares a single PostGIS container per test binary so that
// every integration test of a domain package runs against a migrated database
// without paying the container startup cost each time.
package testutil

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/migrations"
)

var (
	once    sync.Once
	pool    *pgxpool.Pool
	initErr error
	counter atomic.Int64
)

// NewPool returns a pool on a migrated, empty database. The container is
// started on first use and reaped by testcontainers when the test binary
// exits; every call truncates all tables so each test starts from a clean
// state, exactly as if it had its own container. Tests of a package must
// therefore not run in parallel.
func NewPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	once.Do(func() {
		ctx := context.Background()

		container, err := tcpostgres.Run(ctx, "postgis/postgis:16-3.4-alpine",
			tcpostgres.WithDatabase("traily"),
			tcpostgres.WithUsername("traily"),
			tcpostgres.WithPassword("traily"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(30*time.Second),
			),
		)
		if err != nil {
			initErr = fmt.Errorf("start postgres container: %w", err)
			return
		}

		dsn, err := container.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			initErr = fmt.Errorf("get connection string: %w", err)
			return
		}

		pool, err = postgres.Connect(ctx, dsn)
		if err != nil {
			initErr = fmt.Errorf("connect to database: %w", err)
			return
		}

		if err := migrations.Run(ctx, pool); err != nil {
			initErr = fmt.Errorf("run migrations: %w", err)
		}
	})

	if initErr != nil {
		t.Fatalf("test database: %v", initErr)
	}
	truncateAll(t, pool)
	return pool
}

// truncateAll empties every application table, leaving the schema and the
// migration history in place.
func truncateAll(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	const stmt = `
DO $$
DECLARE tables TEXT;
BEGIN
    SELECT string_agg(format('%I.%I', schemaname, tablename), ', ') INTO tables
    FROM pg_tables
    WHERE schemaname = 'public' AND tablename NOT IN ('schema_migrations', 'spatial_ref_sys');
    EXECUTE 'TRUNCATE ' || tables || ' CASCADE';
END $$;`
	if _, err := pool.Exec(context.Background(), stmt); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
}

// Unique returns a value that never repeats within the test binary.
func Unique(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, counter.Add(1))
}

// QueryID runs a statement that RETURNS a single id and returns it. It is how
// tests create the rows an entity depends on without going through its service.
func QueryID(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) string {
	t.Helper()

	var id string
	if err := pool.QueryRow(context.Background(), sql, args...).Scan(&id); err != nil {
		t.Fatalf("QueryID(%q) error = %v", sql, err)
	}
	return id
}

// CreateUser inserts a user with a unique email and returns its id.
func CreateUser(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()

	return QueryID(t, pool,
		`INSERT INTO "user" (email, password, name) VALUES ($1, 'supersecret', 'Test User') RETURNING id`,
		Unique("user")+"@example.com")
}

// CreateTrack inserts a track owned by userID and returns its id.
func CreateTrack(t *testing.T, pool *pgxpool.Pool, userID string) string {
	t.Helper()

	return QueryID(t, pool,
		`INSERT INTO track (user_id, route, distance_m, elevation_gain_m, elevation_loss_m)
		 VALUES ($1, ST_GeomFromText('LINESTRING(0 0, 1 1)', 4326), 0, 0, 0) RETURNING id`,
		userID)
}
