// Package seed fills a development database with sample accounts, content and
// reports, so that the moderation dashboard has something to show. Running it
// again is safe: rows that already exist are left alone.
package seed

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/user"
)

// Password is the password of every seeded account. It is for local
// development only.
const Password = "traily-dev-password"

type account struct {
	email    string
	username string
	name     string
	role     user.Role
}

var accounts = []account{
	{"admin@traily.dev", "admin", "Admin Traily", user.RoleAdmin},
	{"moderator@traily.dev", "moderator", "Modérateur Traily", user.RoleModerator},
	{"alice@traily.dev", "alice", "Alice", user.RoleUser},
	{"bob@traily.dev", "bob", "Bob", user.RoleUser},
	{"charlie@traily.dev", "charlie", "Charlie", user.RoleUser},
}

const (
	insertUserQuery = `
		INSERT INTO users (email, password, name, username, role)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT DO NOTHING`

	insertPhotoQuery = `
		INSERT INTO photos (user_id, storage_key)
		SELECT id, $2 FROM users WHERE email = $1
		ON CONFLICT DO NOTHING`

	insertTrailQuery = `
		INSERT INTO trails (user_id, name, description, distance_meters, elevation_gain_meters, elevation_loss_meters, path)
		SELECT u.id, $2, $3, 4200, 180, 180,
			ST_GeomFromText('LINESTRING(2.2945 48.8584, 2.2990 48.8600, 2.3010 48.8570)', 4326)
		FROM users u
		WHERE u.email = $1
		AND NOT EXISTS (SELECT 1 FROM trails t WHERE t.name = $2)`

	// insertReportQuery adds a report unless the same reporter already made
	// the same report about the same target. The target of a photo or trail
	// report is looked up by its storage key or name; a user report by email.
	insertReportQuery = `
		INSERT INTO reports (reporter_id, target_type, target_id, reason, status, reviewed_by, reviewed_at)
		SELECT reporter.id, $2::text::report_target, $3::uuid, $4, $5::text::report_status,
			CASE WHEN $5 = 'pending' THEN NULL ELSE moderator.id END,
			CASE WHEN $5 = 'pending' THEN NULL ELSE now() END
		FROM users reporter, users moderator
		WHERE reporter.email = $1 AND moderator.email = 'moderator@traily.dev'
		AND NOT EXISTS (
			SELECT 1 FROM reports r
			WHERE r.reporter_id = reporter.id AND r.target_type = $2::text::report_target
			AND r.target_id = $3::uuid AND r.reason = $4
		)`

	userIDQuery  = `SELECT id::text FROM users WHERE email = $1`
	photoIDQuery = `SELECT id::text FROM photos WHERE storage_key = $1`
	trailIDQuery = `SELECT id::text FROM trails WHERE name = $1`
)

type seededReport struct {
	reporterEmail string
	targetType    string
	// targetRef is an email, a storage key or a trail name depending on
	// targetType.
	targetRef string
	reason    string
	status    string
}

var reports = []seededReport{
	{"bob@traily.dev", "photo", "seed/alice-summit.jpg", "Photo qui ne correspond pas à la randonnée, contenu choquant", "pending"},
	{"charlie@traily.dev", "user", "bob@traily.dev", "Commentaires agressifs envers d'autres randonneurs", "pending"},
	{"alice@traily.dev", "trail", "Boucle du lac (seed)", "Le tracé traverse une zone protégée interdite au public", "pending"},
	{"alice@traily.dev", "user", "charlie@traily.dev", "Spam : publicités répétées dans ses contributions", "dismissed"},
}

// Run inserts the sample data. It hashes the account passwords with the same
// cost as the API, so it takes a few seconds the first time.
func Run(ctx context.Context, pool *pgxpool.Pool) error {
	hash, err := user.HashPassword(Password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		for _, a := range accounts {
			if _, err := tx.Exec(ctx, insertUserQuery, a.email, hash, a.name, a.username, string(a.role)); err != nil {
				return fmt.Errorf("seed user %s: %w", a.email, err)
			}
		}

		if _, err := tx.Exec(ctx, insertPhotoQuery, "alice@traily.dev", "seed/alice-summit.jpg"); err != nil {
			return fmt.Errorf("seed photo: %w", err)
		}
		if _, err := tx.Exec(ctx, insertTrailQuery, "bob@traily.dev", "Boucle du lac (seed)", "Parcours d'exemple pour le développement"); err != nil {
			return fmt.Errorf("seed trail: %w", err)
		}

		for _, r := range reports {
			targetID, err := targetID(ctx, tx, r)
			if err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, insertReportQuery, r.reporterEmail, r.targetType, targetID, r.reason, r.status); err != nil {
				return fmt.Errorf("seed report on %s: %w", r.targetRef, err)
			}
		}
		return nil
	})
}

// targetID resolves the target of r to its ID.
func targetID(ctx context.Context, tx pgx.Tx, r seededReport) (string, error) {
	query := userIDQuery
	switch r.targetType {
	case "photo":
		query = photoIDQuery
	case "trail":
		query = trailIDQuery
	}

	var id string
	if err := tx.QueryRow(ctx, query, r.targetRef).Scan(&id); err != nil {
		return "", fmt.Errorf("find %s %q: %w", r.targetType, r.targetRef, err)
	}
	return id, nil
}
