// Package middleware holds the gin middlewares of the API: authentication,
// role checks and rate limiting.
package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/auth"
	"github.com/traily-org/server/internal/domain/user"
)

const (
	userIDKey   = "auth.user_id"
	userRoleKey = "auth.user_role"
)

// SetClaims stores the authenticated user in the request context.
func SetClaims(c *gin.Context, claims auth.Claims) {
	c.Set(userIDKey, claims.UserID)
	c.Set(userRoleKey, claims.Role)
}

// UserID returns the ID of the authenticated user, set by Authenticate.
func UserID(c *gin.Context) (string, bool) {
	id, ok := c.Get(userIDKey)
	if !ok {
		return "", false
	}
	s, ok := id.(string)
	return s, ok && s != ""
}

// UserRole returns the role of the authenticated user, set by Authenticate.
func UserRole(c *gin.Context) (user.Role, bool) {
	role, ok := c.Get(userRoleKey)
	if !ok {
		return "", false
	}
	r, ok := role.(user.Role)
	return r, ok
}
