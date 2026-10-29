package main

import (
	"context"

	"github.com/charmbracelet/log"

	"github.com/traily-org/server/internal/adapters/http/handlers"
	"github.com/traily-org/server/internal/adapters/http/server"
	"github.com/traily-org/server/internal/config"
	"github.com/traily-org/server/internal/domain/activity"
	"github.com/traily-org/server/internal/domain/badge"
	"github.com/traily-org/server/internal/domain/herbier"
	"github.com/traily-org/server/internal/domain/herbierentry"
	"github.com/traily-org/server/internal/domain/moderationlog"
	"github.com/traily-org/server/internal/domain/observation"
	"github.com/traily-org/server/internal/domain/photo"
	"github.com/traily-org/server/internal/domain/plantspecies"
	"github.com/traily-org/server/internal/domain/poi"
	"github.com/traily-org/server/internal/domain/protectedzone"
	"github.com/traily-org/server/internal/domain/publication"
	"github.com/traily-org/server/internal/domain/refreshtoken"
	"github.com/traily-org/server/internal/domain/report"
	"github.com/traily-org/server/internal/domain/species"
	"github.com/traily-org/server/internal/domain/team"
	"github.com/traily-org/server/internal/domain/teammember"
	"github.com/traily-org/server/internal/domain/trace"
	"github.com/traily-org/server/internal/domain/track"
	"github.com/traily-org/server/internal/domain/trail"
	"github.com/traily-org/server/internal/domain/user"
	"github.com/traily-org/server/internal/domain/userbadge"
	"github.com/traily-org/server/internal/domain/userpoint"
	"github.com/traily-org/server/internal/domain/waypoint"
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

	// User service
	userRepository := postgres.NewUserRepository(pool)
	userService := user.NewService(userRepository)
	userHandler := handlers.NewUserHandler(userService)

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

	// Trail service
	trailRepository := postgres.NewTrailRepository(pool)
	trailService := trail.NewService(trailRepository)
	trailHandler := handlers.NewTrailHandler(trailService)

	// Waypoint service
	waypointRepository := postgres.NewWaypointRepository(pool)
	waypointService := waypoint.NewService(waypointRepository)
	waypointHandler := handlers.NewWaypointHandler(waypointService)

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

	// RefreshToken service
	refreshTokenRepository := postgres.NewRefreshTokenRepository(pool)
	refreshTokenService := refreshtoken.NewService(refreshTokenRepository)
	refreshTokenHandler := handlers.NewRefreshTokenHandler(refreshTokenService)

	// UserPoint service
	userPointRepository := postgres.NewUserPointRepository(pool)
	userPointService := userpoint.NewService(userPointRepository)
	userPointHandler := handlers.NewUserPointHandler(userPointService)

	// Badge service
	badgeRepository := postgres.NewBadgeRepository(pool)
	badgeService := badge.NewService(badgeRepository)
	badgeHandler := handlers.NewBadgeHandler(badgeService)

	// UserBadge service
	userBadgeRepository := postgres.NewUserBadgeRepository(pool)
	userBadgeService := userbadge.NewService(userBadgeRepository)
	userBadgeHandler := handlers.NewUserBadgeHandler(userBadgeService)

	// Team service
	teamRepository := postgres.NewTeamRepository(pool)
	teamService := team.NewService(teamRepository)
	teamHandler := handlers.NewTeamHandler(teamService)

	// TeamMember service
	teamMemberRepository := postgres.NewTeamMemberRepository(pool)
	teamMemberService := teammember.NewService(teamMemberRepository)
	teamMemberHandler := handlers.NewTeamMemberHandler(teamMemberService)

	// ProtectedZone service
	protectedZoneRepository := postgres.NewProtectedZoneRepository(pool)
	protectedZoneService := protectedzone.NewService(protectedZoneRepository)
	protectedZoneHandler := handlers.NewProtectedZoneHandler(protectedZoneService)

	// Report service
	reportRepository := postgres.NewReportRepository(pool)
	reportService := report.NewService(reportRepository)
	reportHandler := handlers.NewReportHandler(reportService)

	// ModerationLog service
	moderationLogRepository := postgres.NewModerationLogRepository(pool)
	moderationLogService := moderationlog.NewService(moderationLogRepository)
	moderationLogHandler := handlers.NewModerationLogHandler(moderationLogService)

	srv := server.NewServer(server.Handlers{
		User:          userHandler,
		Track:         trackHandler,
		Trail:         trailHandler,
		Waypoint:      waypointHandler,
		Trace:         traceHandler,
		Activity:      activityHandler,
		Publication:   publicationHandler,
		Poi:           poiHandler,
		Herbier:       herbierHandler,
		PlantSpecies:  plantSpeciesHandler,
		HerbierEntry:  herbierEntryHandler,
		Species:       speciesHandler,
		Photo:         photoHandler,
		Observation:   observationHandler,
		RefreshToken:  refreshTokenHandler,
		UserPoint:     userPointHandler,
		Badge:         badgeHandler,
		UserBadge:     userBadgeHandler,
		Team:          teamHandler,
		TeamMember:    teamMemberHandler,
		ProtectedZone: protectedZoneHandler,
		Report:        reportHandler,
		ModerationLog: moderationLogHandler,
	})
	if err := srv.Run(cfg.Port); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
