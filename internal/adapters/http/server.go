// Package http exposes the Traily REST API over HTTP using gin.
package http

import (
	"fmt"
	nethttp "net/http"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/adapters/http/response"
	"github.com/traily-org/server/internal/pkg/apperror"
)

// codeMethodNotAllowed is the error code returned for 405 responses.
const codeMethodNotAllowed apperror.Code = "METHOD_NOT_ALLOWED"

// Server is the HTTP server of the API.
type Server struct {
	router      *gin.Engine
	userHandler *UserHandler
}

// NewServer builds a Server with its middlewares and routes registered.
func NewServer(userHandler *UserHandler) *Server {
	router := gin.New()
	router.HandleMethodNotAllowed = true
	router.Use(gin.Logger(), recovery())
	router.NoRoute(func(c *gin.Context) {
		response.Error(c, apperror.NotFound("route not found"))
	})
	router.NoMethod(func(c *gin.Context) {
		response.WriteError(c, nethttp.StatusMethodNotAllowed, codeMethodNotAllowed, "method not allowed")
	})

	s := &Server{
		router:      router,
		userHandler: userHandler,
	}
	s.registerRoutes()
	return s
}

// Handler returns the underlying http.Handler, mainly for tests.
func (s *Server) Handler() nethttp.Handler {
	return s.router
}

// Run starts listening on the given port and blocks until the server stops.
func (s *Server) Run(port string) error {
	return s.router.Run(fmt.Sprintf(":%s", port))
}

func (s *Server) registerRoutes() {
	api := s.router.Group("/api")
	v1 := api.Group("/v1")

	v1.GET("/users/:id", s.userHandler.Get)
	v1.POST("/users", s.userHandler.Create)
	v1.PUT("/users/:id", s.userHandler.Update)
	v1.DELETE("/users/:id", s.userHandler.Delete)
}

// recovery returns a middleware that turns panics into a 500 response using
// the error envelope. The panic value is logged, never sent to the client.
func recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		response.Error(c, apperror.Internal(fmt.Errorf("panic recovered: %v", recovered)))
	})
}
