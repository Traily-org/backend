package middleware

import (
	"context"
	nethttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/auth"
	"github.com/traily-org/server/internal/domain/user"
	"github.com/traily-org/server/internal/pkg/apperror"
)

type fakeBanChecker struct {
	banned  map[string]bool
	checked []string
}

func (f *fakeBanChecker) CheckUser(_ context.Context, userID string) error {
	f.checked = append(f.checked, userID)
	if f.banned[userID] {
		return &apperror.Error{Code: apperror.CodeAccountBanned, Message: "account permanently banned"}
	}
	return nil
}

func runRejectBanned(t *testing.T, checker *fakeBanChecker, userID string) (*httptest.ResponseRecorder, bool) {
	t.Helper()

	reached := false
	router := gin.New()
	router.GET("/", func(c *gin.Context) {
		if userID != "" {
			SetClaims(c, auth.Claims{UserID: userID, Role: user.RoleUser})
		}
		c.Next()
	}, RejectBanned(checker), func(c *gin.Context) {
		reached = true
		c.Status(nethttp.StatusOK)
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(nethttp.MethodGet, "/", nil))
	return rec, reached
}

func TestRejectBanned_BannedUser_Returns403AndStops(t *testing.T) {
	checker := &fakeBanChecker{banned: map[string]bool{"u-1": true}}

	rec, reached := runRejectBanned(t, checker, "u-1")

	if rec.Code != nethttp.StatusForbidden || reached {
		t.Fatalf("status = %d, reached = %v, want 403 and the handler not reached", rec.Code, reached)
	}
}

func TestRejectBanned_NotBannedUser_CallsNext(t *testing.T) {
	checker := &fakeBanChecker{banned: map[string]bool{}}

	rec, reached := runRejectBanned(t, checker, "u-2")

	if rec.Code != nethttp.StatusOK || !reached {
		t.Fatalf("status = %d, reached = %v, want 200 and the handler reached", rec.Code, reached)
	}
	if len(checker.checked) != 1 || checker.checked[0] != "u-2" {
		t.Errorf("checked = %v, want [u-2]", checker.checked)
	}
}

func TestRejectBanned_NoAuthenticatedUser_CallsNextWithoutCheck(t *testing.T) {
	checker := &fakeBanChecker{}

	_, reached := runRejectBanned(t, checker, "")

	if !reached || len(checker.checked) != 0 {
		t.Fatalf("reached = %v, checked = %v, want the handler reached without a check", reached, checker.checked)
	}
}
