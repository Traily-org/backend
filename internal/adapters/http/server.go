package http

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

type Server struct {
	router          *gin.Engine
	userHandler     *UserHandler
	appUserHandler  *AppUserHandler
}

func NewServer(userHandler *UserHandler, appUserHandler *AppUserHandler) *Server {
	s := &Server{
		router:         gin.Default(),
		userHandler:    userHandler,
		appUserHandler: appUserHandler,
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
}
