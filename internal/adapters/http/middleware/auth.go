package middleware

import (
	"slices"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/adapters/http/response"
	"github.com/traily-org/server/internal/auth"
	"github.com/traily-org/server/internal/domain/user"
	"github.com/traily-org/server/internal/pkg/apperror"
)

// bearerScheme is the only authorization scheme accepted by Authenticate.
const bearerScheme = "Bearer"

var (
	// errAuthenticationRequired is returned when no usable credentials are sent.
	errAuthenticationRequired = apperror.Unauthorized("authentication required")
	// errInsufficientPermissions is returned when the caller's role is not allowed.
	errInsufficientPermissions = apperror.Forbidden("insufficient permissions")
)

// TokenParser validates an access token and returns the claims it carries.
// It is implemented by *auth.Service.
type TokenParser interface {
	ParseAccessToken(token string) (auth.Claims, error)
}

// Authenticate returns a middleware that requires an
// "Authorization: Bearer <token>" header. The scheme is case-insensitive.
// Missing or malformed headers are answered with 401 UNAUTHORIZED
// "authentication required"; tokens rejected by p are answered with the error
// p returns. On success the claims are stored with SetClaims and the next
// handler runs. The token is never logged.
func Authenticate(p TokenParser) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := bearerToken(c.GetHeader("Authorization"))
		if !ok {
			response.Error(c, errAuthenticationRequired)
			return
		}

		claims, err := p.ParseAccessToken(token)
		if err != nil {
			response.Error(c, err)
			return
		}

		SetClaims(c, claims)
		c.Next()
	}
}

// RequireRole returns a middleware that only lets through authenticated users
// whose role is one of roles. It must run after Authenticate: requests without
// an authenticated user get 401 UNAUTHORIZED, users with another role get
// 403 FORBIDDEN.
func RequireRole(roles ...user.Role) gin.HandlerFunc {
	allowed := slices.Clone(roles)
	return func(c *gin.Context) {
		if _, ok := UserID(c); !ok {
			response.Error(c, errAuthenticationRequired)
			return
		}
		role, ok := UserRole(c)
		if !ok || !slices.Contains(allowed, role) {
			response.Error(c, errInsufficientPermissions)
			return
		}
		c.Next()
	}
}

// bearerToken extracts the token from an Authorization header value of the
// form "Bearer <token>". Whitespace around the whole value is ignored; the
// scheme and the token must be separated by a single space and the token must
// not contain whitespace.
func bearerToken(header string) (string, bool) {
	scheme, token, found := strings.Cut(strings.TrimSpace(header), " ")
	if !found || !strings.EqualFold(scheme, bearerScheme) {
		return "", false
	}
	if token == "" || strings.ContainsAny(token, " \t\r\n") {
		return "", false
	}
	return token, true
}
