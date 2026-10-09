package server

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/adapters/http/handlers"
)

type Server struct {
	router              *gin.Engine
	userHandler         *handlers.UserHandler
	appUserHandler      *handlers.AppUserHandler
	trackHandler        *handlers.TrackHandler
	traceHandler        *handlers.TraceHandler
	activityHandler     *handlers.ActivityHandler
	publicationHandler  *handlers.PublicationHandler
	poiHandler          *handlers.POIHandler
	herbierHandler      *handlers.HerbierHandler
	plantSpeciesHandler *handlers.PlantSpeciesHandler
	herbierEntryHandler *handlers.HerbierEntryHandler
}

func NewServer(
	userHandler *handlers.UserHandler,
	appUserHandler *handlers.AppUserHandler,
	trackHandler *handlers.TrackHandler,
	traceHandler *handlers.TraceHandler,
	activityHandler *handlers.ActivityHandler,
	publicationHandler *handlers.PublicationHandler,
	poiHandler *handlers.POIHandler,
	herbierHandler *handlers.HerbierHandler,
	plantSpeciesHandler *handlers.PlantSpeciesHandler,
	herbierEntryHandler *handlers.HerbierEntryHandler,
) *Server {
	s := &Server{
		router:              gin.Default(),
		userHandler:         userHandler,
		appUserHandler:      appUserHandler,
		trackHandler:        trackHandler,
		traceHandler:        traceHandler,
		activityHandler:     activityHandler,
		publicationHandler:  publicationHandler,
		poiHandler:          poiHandler,
		herbierHandler:      herbierHandler,
		plantSpeciesHandler: plantSpeciesHandler,
		herbierEntryHandler: herbierEntryHandler,
	}
	s.registerRoutes()
	return s
}

func (s *Server) Run(port string) error {
	return s.router.Run(fmt.Sprintf(":%s", port))
}

func (s *Server) registerRoutes() {
	api := s.router.Group("/api")
	v1 := api.Group("/v1")

	// User routes (legacy)
	v1.GET("/users/:id", s.userHandler.Get)
	v1.POST("/users", s.userHandler.Create)
	v1.PUT("/users/:id", s.userHandler.Update)
	v1.DELETE("/users/:id", s.userHandler.Delete)

	// AppUser routes
	v1.GET("/app-users", s.appUserHandler.List)
	v1.POST("/app-users", s.appUserHandler.Create)
	v1.GET("/app-users/:id", s.appUserHandler.Get)
	v1.PUT("/app-users/:id", s.appUserHandler.Update)
	v1.DELETE("/app-users/:id", s.appUserHandler.Delete)

	// Track routes
	v1.GET("/tracks", s.trackHandler.ListByUser)
	v1.POST("/tracks", s.trackHandler.Create)
	v1.GET("/tracks/:id", s.trackHandler.Get)
	v1.PUT("/tracks/:id", s.trackHandler.Update)
	v1.DELETE("/tracks/:id", s.trackHandler.Delete)

	// Trace routes
	v1.GET("/traces", s.traceHandler.List)
	v1.POST("/traces", s.traceHandler.Create)
	v1.GET("/traces/:id", s.traceHandler.Get)
	v1.PUT("/traces/:id", s.traceHandler.Update)
	v1.DELETE("/traces/:id", s.traceHandler.Delete)

	// Activity routes
	v1.GET("/activities", s.activityHandler.List)
	v1.POST("/activities", s.activityHandler.Create)
	v1.GET("/activities/:id", s.activityHandler.Get)
	v1.PUT("/activities/:id", s.activityHandler.Update)
	v1.DELETE("/activities/:id", s.activityHandler.Delete)

	// Publication routes
	v1.GET("/publications", s.publicationHandler.List)
	v1.POST("/publications", s.publicationHandler.Create)
	v1.GET("/publications/:id", s.publicationHandler.Get)
	v1.PUT("/publications/:id", s.publicationHandler.Update)
	v1.DELETE("/publications/:id", s.publicationHandler.Delete)

	// POI routes
	v1.GET("/pois", s.poiHandler.List)
	v1.POST("/pois", s.poiHandler.Create)
	v1.GET("/pois/:id", s.poiHandler.Get)
	v1.PUT("/pois/:id", s.poiHandler.Update)
	v1.DELETE("/pois/:id", s.poiHandler.Delete)

	// Herbier routes
	v1.GET("/herbiers", s.herbierHandler.List)
	v1.POST("/herbiers", s.herbierHandler.Create)
	v1.GET("/herbiers/:id", s.herbierHandler.Get)
	v1.PUT("/herbiers/:id", s.herbierHandler.Update)
	v1.DELETE("/herbiers/:id", s.herbierHandler.Delete)

	// PlantSpecies routes
	v1.GET("/plant-species", s.plantSpeciesHandler.List)
	v1.POST("/plant-species", s.plantSpeciesHandler.Create)
	v1.GET("/plant-species/:id", s.plantSpeciesHandler.Get)
	v1.PUT("/plant-species/:id", s.plantSpeciesHandler.Update)
	v1.DELETE("/plant-species/:id", s.plantSpeciesHandler.Delete)

	// HerbierEntry routes
	v1.GET("/herbier-entries", s.herbierEntryHandler.List)
	v1.POST("/herbier-entries", s.herbierEntryHandler.Create)
	v1.GET("/herbier-entries/:id", s.herbierEntryHandler.Get)
	v1.PUT("/herbier-entries/:id", s.herbierEntryHandler.Update)
	v1.DELETE("/herbier-entries/:id", s.herbierEntryHandler.Delete)
}
