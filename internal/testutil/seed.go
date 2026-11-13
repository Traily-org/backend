//go:build integration

package testutil

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/user"
)

// SeedPassword est le mot de passe de tous les comptes créés par SeedAccount.
const SeedPassword = "seed-password-123"

// Account est un compte créé par SeedAccount, utilisable pour se connecter
// via l'API avec Email et SeedPassword.
type Account struct {
	ID    string
	Email string
	Role  user.Role
}

// SeedAccount insère un compte avec un vrai hash bcrypt de SeedPassword et le
// rôle donné, que l'API ne permet pas de choisir à l'inscription.
func SeedAccount(t *testing.T, pool *pgxpool.Pool, email string, role user.Role) Account {
	t.Helper()

	hash, err := user.HashPassword(SeedPassword)
	if err != nil {
		t.Fatalf("testutil: hash seed password: %v", err)
	}

	email = user.NormalizeEmail(email)
	var id string
	err = pool.QueryRow(context.Background(), `
		INSERT INTO users (email, password, name, role)
		VALUES ($1, $2, 'Seed User', $3)
		RETURNING id::text`, email, hash, string(role)).Scan(&id)
	if err != nil {
		t.Fatalf("testutil: seed account %s: %v", email, err)
	}
	return Account{ID: id, Email: email, Role: role}
}

// SeedReport insère un signalement en attente de reporterID sur la cible
// (target_type parmi photo, observation, trail, user) et retourne son ID.
func SeedReport(t *testing.T, pool *pgxpool.Pool, reporterID, targetType, targetID string) string {
	t.Helper()

	var id string
	err := pool.QueryRow(context.Background(), `
		INSERT INTO reports (reporter_id, target_type, target_id, reason)
		VALUES ($1, $2, $3, 'seeded report')
		RETURNING id::text`, reporterID, targetType, targetID).Scan(&id)
	if err != nil {
		t.Fatalf("testutil: seed report: %v", err)
	}
	return id
}

// SeedPhoto insère une photo de userID et retourne son ID, pour signaler un
// contenu plutôt qu'un compte.
func SeedPhoto(t *testing.T, pool *pgxpool.Pool, userID string) string {
	t.Helper()

	var id string
	err := pool.QueryRow(context.Background(), `
		INSERT INTO photos (user_id, storage_key)
		VALUES ($1, 'seed/' || gen_random_uuid() || '.jpg')
		RETURNING id::text`, userID).Scan(&id)
	if err != nil {
		t.Fatalf("testutil: seed photo: %v", err)
	}
	return id
}

// SeedExpiredBan insère un ban de userID déjà terminé : créé il y a deux
// heures, fini il y a une heure. L'API ne peut pas créer de ban dans le passé.
func SeedExpiredBan(t *testing.T, pool *pgxpool.Pool, userID string) {
	t.Helper()

	_, err := pool.Exec(context.Background(), `
		INSERT INTO user_bans (user_id, reason, expires_at, created_at)
		VALUES ($1, 'seeded expired ban', now() - interval '1 hour', now() - interval '2 hours')`, userID)
	if err != nil {
		t.Fatalf("testutil: seed expired ban: %v", err)
	}
}
