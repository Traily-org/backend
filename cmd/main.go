package main

import (
	"context"

	"github.com/charmbracelet/log"

	"github.com/traily-org/server/internal/adapters/http"
	"github.com/traily-org/server/internal/config"
	"github.com/traily-org/server/internal/domain/appuser"
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

	// Legacy user service
	userRepository := postgres.NewUserRepository(pool)
	userService := user.NewService(userRepository)
	userHandler := http.NewUserHandler(userService)

	// AppUser service
	appUserRepository := postgres.NewAppUserRepository(pool)
	appUserService := appuser.NewService(appUserRepository)
	appUserHandler := http.NewAppUserHandler(appUserService)

	server := http.NewServer(userHandler, appUserHandler)

	if err := server.Run(cfg.Port); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
