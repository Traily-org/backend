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

// UserRoutes are the /users endpoints.
type UserRoutes interface {
	Get(c *gin.Context)
	GetMe(c *gin.Context)
	UpdateMe(c *gin.Context)
	DeleteMe(c *gin.Context)
}

// AuthRoutes are the /auth endpoints.
type AuthRoutes interface {
	Register(c *gin.Context)
	Login(c *gin.Context)
	Refresh(c *gin.Context)
	Logout(c *gin.Context)
}

// Deps are the handlers and middlewares the server is built from.
type Deps struct {
	Users UserRoutes
	Auth  AuthRoutes
	// Authenticate rejects requests without a valid access token.
	Authenticate gin.HandlerFunc
	// AuthRateLimit throttles /auth/register and /auth/login, the endpoints
	// exposed to credential guessing.
	AuthRateLimit gin.HandlerFunc
	// SessionRateLimit throttles /auth/refresh and /auth/logout, called by
	// every signed-in client. Nil means no limit.
	SessionRateLimit gin.HandlerFunc
	// TrustedProxies lists the proxies whose X-Forwarded-For header is
	// trusted to find the client IP. Empty means none: the client IP is the
	// direct peer, so clients cannot spoof it to bypass the rate limit.
	TrustedProxies []string
}

// Server is the HTTP server of the API.
type Server struct {
	router *gin.Engine
	deps   Deps
}

// NewServer builds a Server with its middlewares and routes registered.
// It fails if a trusted proxy is not a valid IP or CIDR.
func NewServer(deps Deps) (*Server, error) {
	router := gin.New()
	if err := router.SetTrustedProxies(deps.TrustedProxies); err != nil {
		return nil, fmt.Errorf("set trusted proxies: %w", err)
	}
	router.HandleMethodNotAllowed = true
	router.Use(gin.Logger(), recovery())
	router.NoRoute(func(c *gin.Context) {
		response.Error(c, apperror.NotFound("route not found"))
	})
	router.NoMethod(func(c *gin.Context) {
		response.WriteError(c, nethttp.StatusMethodNotAllowed, codeMethodNotAllowed, "method not allowed")
	})

	s := &Server{
		router: router,
		deps:   deps,
	}
	s.registerRoutes()
	return s, nil
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

	d := s.deps

	authGroup := v1.Group("/auth")
	authGroup.POST("/register", orNext(d.AuthRateLimit), d.Auth.Register)
	authGroup.POST("/login", orNext(d.AuthRateLimit), d.Auth.Login)
	authGroup.POST("/refresh", orNext(d.SessionRateLimit), d.Auth.Refresh)
	authGroup.POST("/logout", orNext(d.SessionRateLimit), d.Auth.Logout)

	users := v1.Group("/users", d.Authenticate)
	users.GET("/me", d.Users.GetMe)
	users.PUT("/me", d.Users.UpdateMe)
	users.DELETE("/me", d.Users.DeleteMe)
	users.GET("/:id", d.Users.Get)
}

// recovery returns a middleware that turns panics into a 500 response using
// the error envelope. The panic value is logged, never sent to the client.
func recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		response.Error(c, apperror.Internal(fmt.Errorf("panic recovered: %v", recovered)))
	})
}

// orNext returns h, or a middleware that does nothing when h is nil.
func orNext(h gin.HandlerFunc) gin.HandlerFunc {
	if h == nil {
		return func(c *gin.Context) { c.Next() }
	}
	return h
}
