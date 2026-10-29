package server

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/adapters/http/handlers"
)

type Handlers struct {
	User          *handlers.UserHandler
	Track         *handlers.TrackHandler
	Trail         *handlers.TrailHandler
	Waypoint      *handlers.WaypointHandler
	Trace         *handlers.TraceHandler
	Activity      *handlers.ActivityHandler
	Publication   *handlers.PublicationHandler
	Poi           *handlers.PoiHandler
	Herbier       *handlers.HerbierHandler
	PlantSpecies  *handlers.PlantspeciesHandler
	HerbierEntry  *handlers.HerbierentryHandler
	Species       *handlers.SpeciesHandler
	Photo         *handlers.PhotoHandler
	Observation   *handlers.ObservationHandler
	RefreshToken  *handlers.RefreshTokenHandler
	UserPoint     *handlers.UserPointHandler
	Badge         *handlers.BadgeHandler
	UserBadge     *handlers.UserBadgeHandler
	Team          *handlers.TeamHandler
	TeamMember    *handlers.TeamMemberHandler
	ProtectedZone *handlers.ProtectedZoneHandler
	Report        *handlers.ReportHandler
	ModerationLog *handlers.ModerationLogHandler
}

type Server struct {
	router   *gin.Engine
	handlers Handlers
}

func NewServer(h Handlers) *Server {
	s := &Server{
		router:   gin.Default(),
		handlers: h,
	}
	s.registerRoutes()
	return s
}

func (s *Server) Run(port string) error {
	return s.router.Run(fmt.Sprintf(":%s", port))
}

type crudHandler interface {
	List(ctx *gin.Context)
	Create(ctx *gin.Context)
	Get(ctx *gin.Context)
	Update(ctx *gin.Context)
	Delete(ctx *gin.Context)
}

func registerCRUD(group *gin.RouterGroup, path string, h crudHandler) {
	group.GET(path, h.List)
	group.POST(path, h.Create)
	group.GET(path+"/:id", h.Get)
	group.PUT(path+"/:id", h.Update)
	group.DELETE(path+"/:id", h.Delete)
}

func (s *Server) registerRoutes() {
	api := s.router.Group("/api")
	v1 := api.Group("/v1")
	h := s.handlers

	// User routes
	v1.GET("/users/:id", h.User.Get)
	v1.POST("/users", h.User.Create)
	v1.PUT("/users/:id", h.User.Update)
	v1.DELETE("/users/:id", h.User.Delete)

	// Track routes
	v1.GET("/tracks", h.Track.ListByUser)
	v1.POST("/tracks", h.Track.Create)
	v1.GET("/tracks/:id", h.Track.Get)
	v1.PUT("/tracks/:id", h.Track.Update)
	v1.DELETE("/tracks/:id", h.Track.Delete)

	registerCRUD(v1, "/trails", h.Trail)
	registerCRUD(v1, "/waypoints", h.Waypoint)
	registerCRUD(v1, "/traces", h.Trace)
	registerCRUD(v1, "/activities", h.Activity)
	registerCRUD(v1, "/publications", h.Publication)
	registerCRUD(v1, "/pois", h.Poi)
	registerCRUD(v1, "/herbiers", h.Herbier)
	registerCRUD(v1, "/plant-species", h.PlantSpecies)
	registerCRUD(v1, "/herbier-entries", h.HerbierEntry)
	registerCRUD(v1, "/species", h.Species)
	registerCRUD(v1, "/photos", h.Photo)
	registerCRUD(v1, "/observations", h.Observation)
	registerCRUD(v1, "/refresh-tokens", h.RefreshToken)
	registerCRUD(v1, "/user-points", h.UserPoint)
	registerCRUD(v1, "/badges", h.Badge)
	registerCRUD(v1, "/user-badges", h.UserBadge)
	registerCRUD(v1, "/teams", h.Team)
	registerCRUD(v1, "/team-members", h.TeamMember)
	registerCRUD(v1, "/protected-zones", h.ProtectedZone)
	registerCRUD(v1, "/reports", h.Report)
	registerCRUD(v1, "/moderation-logs", h.ModerationLog)
}
