// Command seed fills the development database with sample data. It never runs
// when APP_ENV is "production".
package main

import (
	"context"
	"os"

	"github.com/charmbracelet/log"

	"github.com/traily-org/server/internal/config"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/internal/seed"
	"github.com/traily-org/server/migrations"
)

func main() {
	if os.Getenv("APP_ENV") == "production" {
		log.Fatal("refusing to seed a production database (APP_ENV=production)")
	}

	cfg := config.Load()

	ctx := context.Background()
	pool, err := postgres.Connect(ctx, cfg.DB.DSN())
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := migrations.Run(ctx, pool); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	if err := seed.Run(ctx, pool); err != nil {
		log.Fatalf("failed to seed database: %v", err)
	}

	log.Info("database seeded", "accounts", "admin, moderator, alice, bob, charlie @traily.dev", "password", "see internal/seed")
}
