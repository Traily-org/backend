package main

import (
	"context"

	"github.com/charmbracelet/log"

	"github.com/traily-org/server/internal/adapters/http/handlers"
	"github.com/traily-org/server/internal/adapters/http/server"
	"github.com/traily-org/server/internal/config"
	"github.com/traily-org/server/internal/domain/activity"
	"github.com/traily-org/server/internal/domain/appuser"
	"github.com/traily-org/server/internal/domain/observation"
	"github.com/traily-org/server/internal/domain/photo"
	"github.com/traily-org/server/internal/domain/poi"
	"github.com/traily-org/server/internal/domain/publication"
	"github.com/traily-org/server/internal/domain/species"
	"github.com/traily-org/server/internal/domain/trace"
	"github.com/traily-org/server/internal/domain/track"
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
	userHandler := handlers.NewUserHandler(userService)

	// AppUser service
	appUserRepository := postgres.NewAppUserRepository(pool)
	appUserService := appuser.NewService(appUserRepository)
	appUserHandler := handlers.NewAppUserHandler(appUserService)

	// Track service
	trackRepository := postgres.NewTrackRepository(pool)
	trackService := track.NewService(trackRepository)
	trackHandler := handlers.NewTrackHandler(trackService)

	// Trace service
	traceRepository := postgres.NewTraceRepository(pool)
	traceService := trace.NewService(traceRepository)
	traceHandler := handlers.NewTraceHandler(traceService)

	// Activity service
	activityRepository := postgres.NewActivityRepository(pool)
	activityService := activity.NewService(activityRepository)
	activityHandler := handlers.NewActivityHandler(activityService)

	// Publication service
	publicationRepository := postgres.NewPublicationRepository(pool)
	publicationService := publication.NewService(publicationRepository)
	publicationHandler := handlers.NewPublicationHandler(publicationService)

	// POI service
	poiRepository := postgres.NewPOIRepository(pool)
	poiService := poi.NewService(poiRepository)
	poiHandler := handlers.NewPoiHandler(poiService)

	// Species service
	speciesRepository := postgres.NewSpeciesRepository(pool)
	speciesService := species.NewService(speciesRepository)
	speciesHandler := handlers.NewSpeciesHandler(speciesService)

	// Photo service
	photoRepository := postgres.NewPhotoRepository(pool)
	photoService := photo.NewService(photoRepository)
	photoHandler := handlers.NewPhotoHandler(photoService)

	// Observation service
	observationRepository := postgres.NewObservationRepository(pool)
	observationService := observation.NewService(observationRepository)
	observationHandler := handlers.NewObservationHandler(observationService)

	srv := server.NewServer(
		userHandler,
		appUserHandler,
		trackHandler,
		traceHandler,
		activityHandler,
		publicationHandler,
		poiHandler,
		speciesHandler,
		photoHandler,
		observationHandler,
	)

	if err := srv.Run(cfg.Port); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
