package main

import (
	"context"

	"github.com/charmbracelet/log"

	"github.com/traily-org/server/internal/adapters/http/handlers"
	"github.com/traily-org/server/internal/adapters/http/server"
	"github.com/traily-org/server/internal/config"
	"github.com/traily-org/server/internal/domain/activity"
	"github.com/traily-org/server/internal/domain/appuser"
	"github.com/traily-org/server/internal/domain/herbier"
	"github.com/traily-org/server/internal/domain/herbierentry"
	"github.com/traily-org/server/internal/domain/plantspecies"
	"github.com/traily-org/server/internal/domain/poi"
	"github.com/traily-org/server/internal/domain/publication"
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

	// Herbier service
	herbierRepository := postgres.NewHerbierRepository(pool)
	herbierService := herbier.NewService(herbierRepository)
	herbierHandler := handlers.NewHerbierHandler(herbierService)

	// PlantSpecies service
	plantSpeciesRepository := postgres.NewPlantSpeciesRepository(pool)
	plantSpeciesService := plantspecies.NewService(plantSpeciesRepository)
	plantSpeciesHandler := handlers.NewPlantspeciesHandler(plantSpeciesService)

	// HerbierEntry service
	herbierEntryRepository := postgres.NewHerbierEntryRepository(pool)
	herbierEntryService := herbierentry.NewService(herbierEntryRepository)
	herbierEntryHandler := handlers.NewHerbierentryHandler(herbierEntryService)

	srv := server.NewServer(
		userHandler,
		appUserHandler,
		trackHandler,
		traceHandler,
		activityHandler,
		publicationHandler,
		poiHandler,
		herbierHandler,
		plantSpeciesHandler,
		herbierEntryHandler,
	)

	if err := srv.Run(cfg.Port); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
