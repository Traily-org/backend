package middleware

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/adapters/http/response"
)

// UserBanChecker returns nil when the user is not banned, and the error to
// give back to the client when they are. It is implemented by *ban.Service.
type UserBanChecker interface {
	CheckUser(ctx context.Context, userID string) error
}

// RejectBanned returns a middleware that refuses every request of a banned
// user with the checker's error, so that an access token issued before the
// ban stops working at once instead of at its expiry. It costs one indexed
// query per authenticated request. It must run after Authenticate; requests
// without an authenticated user are let through for the next guard to judge.
func RejectBanned(b UserBanChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := UserID(c)
		if !ok {
			c.Next()
			return
		}

		if err := b.CheckUser(c.Request.Context(), id); err != nil {
			response.Error(c, err)
			return
		}
		c.Next()
	}
}
