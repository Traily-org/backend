//go:build integration

package seed_test

import (
	"context"
	"testing"

	"github.com/traily-org/server/internal/seed"
	"github.com/traily-org/server/internal/testutil"
)

func TestRun_IsIdempotentAndCreatesPendingReports(t *testing.T) {
	pool := testutil.NewPool(t)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		if err := seed.Run(ctx, pool); err != nil {
			t.Fatalf("Run() #%d error = %v", i+1, err)
		}
	}

	counts := map[string]int{"users": 5, "photos": 1, "trails": 1, "reports": 4}
	for table, want := range counts {
		var got int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&got); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if got != want {
			t.Errorf("%s = %d after two runs, want %d", table, got, want)
		}
	}

	var pending, reviewedWithModerator int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE status = 'pending'),
		       count(*) FILTER (WHERE status <> 'pending' AND reviewed_by IS NOT NULL AND reviewed_at IS NOT NULL)
		FROM reports`).Scan(&pending, &reviewedWithModerator); err != nil {
		t.Fatalf("count reports: %v", err)
	}
	if pending != 3 || reviewedWithModerator != 1 {
		t.Errorf("pending = %d, handled with a moderator = %d, want 3 and 1", pending, reviewedWithModerator)
	}

	var role string
	if err := pool.QueryRow(ctx, `SELECT role FROM users WHERE email = 'moderator@traily.dev'`).Scan(&role); err != nil || role != "moderator" {
		t.Errorf("moderator role = %q, %v, want moderator", role, err)
	}
}
