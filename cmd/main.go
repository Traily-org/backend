package main

import (
	"context"

	"github.com/charmbracelet/log"

	"github.com/traily-org/server/internal/adapters/http"
	"github.com/traily-org/server/internal/config"
	"github.com/traily-org/server/internal/domain/trail"
	"github.com/traily-org/server/internal/domain/user"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/migrations"
)

func main() {
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

	userRepository := postgres.NewUserRepository(pool)
	userService := user.NewService(userRepository)
	userHandler := http.NewUserHandler(userService)

	trailRepository := postgres.NewTrailRepository(pool)
	trailService := trail.NewService(trailRepository)
	trailHandler := http.NewTrailHandler(trailService)

	server := http.NewServer(userHandler, trailHandler)

	if err := server.Run(cfg.Port); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
