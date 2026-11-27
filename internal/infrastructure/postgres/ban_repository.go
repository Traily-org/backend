package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/ban"
)

const (
	banColumns = `id::text, user_id::text, report_id::text, reason, banned_by::text, expires_at, lifted_at, lifted_by::text, created_at`

	// getReportQuery returns a report with the user it concerns: the
	// reported user, or the author of the reported content.
	getReportQuery = `
		SELECT r.id::text,
			CASE r.target_type
				WHEN 'user' THEN (SELECT u.id FROM users u WHERE u.id = r.target_id)
				WHEN 'photo' THEN (SELECT p.user_id FROM photos p WHERE p.id = r.target_id)
				WHEN 'observation' THEN (SELECT o.user_id FROM observations o WHERE o.id = r.target_id)
				WHEN 'trail' THEN (SELECT t.user_id FROM trails t WHERE t.id = r.target_id)
			END::text,
			r.status::text
		FROM reports r
		WHERE r.id = $1`

	reviewReportQuery = `
		UPDATE reports
		SET status = 'reviewed', reviewed_by = $2, reviewed_at = now()
		WHERE id = $1`

	// activeBanCondition selects the bans in force at $2 for the user $1.
	activeBanCondition = `user_id = $1 AND lifted_at IS NULL AND (expires_at IS NULL OR expires_at > $2)`

	createBanQuery = `
		INSERT INTO user_bans (user_id, report_id, reason, banned_by, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING ` + banColumns

	// banIdentifierQuery bans an identifier. When it is already banned, the
	// latest expiry wins and NULL (permanent) beats any date.
	banIdentifierQuery = `
		INSERT INTO banned_identifiers (type, value_hash, user_ban_id, expires_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (type, value_hash) DO UPDATE SET
			user_ban_id = EXCLUDED.user_ban_id,
			expires_at = CASE
				WHEN banned_identifiers.expires_at IS NULL OR EXCLUDED.expires_at IS NULL THEN NULL
				ELSE GREATEST(banned_identifiers.expires_at, EXCLUDED.expires_at)
			END`

	getActiveBanQuery = `
		SELECT ` + banColumns + `
		FROM user_bans
		WHERE ` + activeBanCondition + `
		ORDER BY expires_at DESC NULLS FIRST, created_at DESC
		LIMIT 1`

	// liftBansQuery lifts the active bans of $1 at $2 by $3, removes their
	// banned identifiers and returns how many bans were lifted.
	liftBansQuery = `
		WITH lifted AS (
			UPDATE user_bans
			SET lifted_at = $2, lifted_by = $3
			WHERE ` + activeBanCondition + `
			RETURNING id
		), unbanned AS (
			DELETE FROM banned_identifiers
			WHERE user_ban_id IN (SELECT id FROM lifted)
		)
		SELECT count(*) FROM lifted`

	identifierBannedQuery = `
		SELECT EXISTS (
			SELECT 1
			FROM banned_identifiers
			WHERE type = $1 AND value_hash = $2 AND (expires_at IS NULL OR expires_at > $3)
		)`

	logModerationQuery = `
		INSERT INTO moderation_logs (moderator_id, action, target_type, target_id, report_id, notes)
		VALUES ($1, $2, 'user', $3, $4, $5)`
)

// BanRepository is the PostgreSQL implementation of ban.Repository.
type BanRepository struct {
	pool *pgxpool.Pool
}

// NewBanRepository returns a BanRepository backed by the given pool.
func NewBanRepository(pool *pgxpool.Pool) *BanRepository {
	return &BanRepository{pool: pool}
}

// Report returns the report identified by id with the user it concerns, or
// ban.ErrReportNotFound.
func (r *BanRepository) Report(ctx context.Context, id string) (ban.Report, error) {
	var rep ban.Report
	var status string
	err := r.pool.QueryRow(ctx, getReportQuery, id).Scan(&rep.ID, &rep.TargetUserID, &status)
	if err != nil {
		return ban.Report{}, mapError(err, ban.ErrReportNotFound)
	}
	rep.Status = ban.ReportStatus(status)
	return rep, nil
}

// Create stores b, bans ids, marks the report as reviewed and logs the ban,
// in one transaction.
func (r *BanRepository) Create(ctx context.Context, b ban.Ban, ids []ban.Identifier) (ban.Ban, error) {
	var created ban.Ban
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		var err error
		created, err = scanBan(tx.QueryRow(ctx, createBanQuery, b.UserID, b.ReportID, b.Reason, b.BannedBy, b.ExpiresAt))
		if err != nil {
			return err
		}

		for _, id := range ids {
			if _, err := tx.Exec(ctx, banIdentifierQuery, string(id.Type), id.ValueHash, created.ID, id.ExpiresAt); err != nil {
				return err
			}
		}

		if _, err := tx.Exec(ctx, reviewReportQuery, b.ReportID, b.BannedBy); err != nil {
			return err
		}

		_, err = tx.Exec(ctx, logModerationQuery, b.BannedBy, "ban", b.UserID, b.ReportID, b.Reason)
		return err
	})
	if err != nil {
		return ban.Ban{}, mapError(err, ban.ErrNotBanned)
	}
	return created, nil
}

// Active returns the ban of userID active at at that ends last, or
// ban.ErrNotBanned.
func (r *BanRepository) Active(ctx context.Context, userID string, at time.Time) (ban.Ban, error) {
	b, err := scanBan(r.pool.QueryRow(ctx, getActiveBanQuery, userID, at))
	if err != nil {
		return ban.Ban{}, mapError(err, ban.ErrNotBanned)
	}
	return b, nil
}

// Lift lifts the active bans of userID, removes their banned identifiers and
// logs it when at least one ban was lifted, in one transaction.
func (r *BanRepository) Lift(ctx context.Context, userID, liftedBy string, at time.Time) (int, error) {
	var n int
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, liftBansQuery, userID, at, liftedBy).Scan(&n); err != nil {
			return err
		}
		if n == 0 {
			return nil
		}
		_, err := tx.Exec(ctx, logModerationQuery, liftedBy, "unban", userID, nil, nil)
		return err
	})
	if err != nil {
		return 0, mapError(err, ban.ErrNotBanned)
	}
	return n, nil
}

// IdentifierBanned reports whether the identifier is banned at at.
func (r *BanRepository) IdentifierBanned(ctx context.Context, kind ban.IdentifierType, hash string, at time.Time) (bool, error) {
	var banned bool
	if err := r.pool.QueryRow(ctx, identifierBannedQuery, string(kind), hash, at).Scan(&banned); err != nil {
		return false, mapError(err, ban.ErrNotBanned)
	}
	return banned, nil
}

// scanBan reads a row selected with banColumns.
func scanBan(row pgx.Row) (ban.Ban, error) {
	var b ban.Ban
	err := row.Scan(&b.ID, &b.UserID, &b.ReportID, &b.Reason, &b.BannedBy, &b.ExpiresAt, &b.LiftedAt, &b.LiftedBy, &b.CreatedAt)
	return b, err
}
