package main

import (
	"context"
	"time"

	"github.com/charmbracelet/log"

	"github.com/traily-org/server/internal/adapters/http"
	"github.com/traily-org/server/internal/adapters/http/middleware"
	"github.com/traily-org/server/internal/auth"
	"github.com/traily-org/server/internal/config"
	"github.com/traily-org/server/internal/domain/user"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/migrations"
)

// Rate limits applied to each client IP: strict on register and login, which
// are exposed to credential guessing, looser on refresh and logout, which
// every signed-in client calls and many may share a mobile carrier IP.
const (
	authRateLimitRequests    = 10
	sessionRateLimitRequests = 60
	rateLimitWindow          = time.Minute
)

func main() {
	cfg := config.Load()
	if err := cfg.Auth.Validate(); err != nil {
		log.Fatalf("invalid auth configuration: %v", err)
	}

	ctx := context.Background()
	pool, err := postgres.Connect(ctx, cfg.DB.DSN())
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := migrations.Run(ctx, pool); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	userService := user.NewService(postgres.NewUserRepository(pool))
	authService := auth.NewService(userService, postgres.NewRefreshTokenRepository(pool), auth.Config{
		AccessSecret:  []byte(cfg.Auth.AccessSecret),
		RefreshSecret: []byte(cfg.Auth.RefreshSecret),
		AccessTTL:     cfg.Auth.AccessTTL,
		RefreshTTL:    cfg.Auth.RefreshTTL,
		Issuer:        "traily",
	})
	userService.SetSessionRevoker(authService)

	server, err := http.NewServer(http.Deps{
		Users:            http.NewUserHandler(userService),
		Auth:             http.NewAuthHandler(authService),
		Authenticate:     middleware.Authenticate(authService),
		AuthRateLimit:    middleware.RateLimit(authRateLimitRequests, rateLimitWindow),
		SessionRateLimit: middleware.RateLimit(sessionRateLimitRequests, rateLimitWindow),
		TrustedProxies:   cfg.TrustedProxies,
	})
	if err != nil {
		log.Fatalf("failed to build server: %v", err)
	}

	if err := server.Run(cfg.Port); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
