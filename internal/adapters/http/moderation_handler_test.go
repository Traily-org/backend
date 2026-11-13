package http

import (
	"context"
	"encoding/json"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/adapters/http/middleware"
	"github.com/traily-org/server/internal/auth"
	"github.com/traily-org/server/internal/domain/ban"
	"github.com/traily-org/server/internal/domain/user"
	"github.com/traily-org/server/internal/pkg/apperror"
)

// testRoleHeader carries the role of the authenticated user in moderation
// handler tests.
const testRoleHeader = "X-Test-Role"

type fakeBanService struct {
	banFn    func(ctx context.Context, in ban.Input) (ban.Ban, error)
	liftFn   func(ctx context.Context, userID string, by ban.Moderator) error
	activeFn func(ctx context.Context, userID string) (ban.Ban, error)
}

func (f *fakeBanService) Ban(ctx context.Context, in ban.Input) (ban.Ban, error) {
	return f.banFn(ctx, in)
}

func (f *fakeBanService) Lift(ctx context.Context, userID string, by ban.Moderator) error {
	return f.liftFn(ctx, userID, by)
}

func (f *fakeBanService) Active(ctx context.Context, userID string) (ban.Ban, error) {
	return f.activeFn(ctx, userID)
}

// testAuthenticateWithRole is testAuthenticate with the role read from
// testRoleHeader.
func testAuthenticateWithRole(c *gin.Context) {
	if id := c.GetHeader(testUserHeader); id != "" {
		middleware.SetClaims(c, auth.Claims{UserID: id, Role: user.Role(c.GetHeader(testRoleHeader))})
	}
	c.Next()
}

func serveModeration(t *testing.T, svc *fakeBanService, method, path, body, userID string, role user.Role) (*httptest.ResponseRecorder, envelope) {
	t.Helper()

	srv, err := NewServer(Deps{
		Users:            NewUserHandler(&fakeUserService{}),
		Auth:             noopAuthRoutes{},
		Moderation:       NewModerationHandler(svc),
		Authenticate:     testAuthenticateWithRole,
		RequireModerator: middleware.RequireRole(user.RoleModerator, user.RoleAdmin),
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if userID != "" {
		req.Header.Set(testUserHeader, userID)
		req.Header.Set(testRoleHeader, string(role))
	}
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	var env envelope
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
			t.Fatalf("invalid JSON body %q: %v", rec.Body.String(), err)
		}
	}
	return rec, env
}

func TestModerationHandler_BanUser_Moderator_Returns201WithBan(t *testing.T) {
	expires := time.Date(2026, 11, 20, 14, 0, 0, 0, paris)
	svc := &fakeBanService{banFn: func(_ context.Context, in ban.Input) (ban.Ban, error) {
		if in.ReportID != "11111111-1111-4111-8111-111111111111" {
			t.Errorf("ReportID = %q", in.ReportID)
		}
		if in.UserID != "u-2" || in.Reason != "spam" || in.By != (ban.Moderator{ID: "mod-1", Role: user.RoleModerator}) {
			t.Errorf("Ban() input = %+v", in)
		}
		if in.ExpiresAt == nil || !in.ExpiresAt.Equal(expires) {
			t.Errorf("ExpiresAt = %v, want %v", in.ExpiresAt, expires)
		}
		return ban.Ban{ID: "b-1", UserID: in.UserID, ReportID: ptr(in.ReportID), Reason: in.Reason, BannedBy: ptr("mod-1"), ExpiresAt: in.ExpiresAt, CreatedAt: createdAt}, nil
	}}

	rec, env := serveModeration(t, svc, nethttp.MethodPost, "/api/v1/moderation/users/u-2/ban",
		`{"report_id":"11111111-1111-4111-8111-111111111111","reason":"spam","expires_at":"2026-11-20T14:00:00+02:00"}`, "mod-1", user.RoleModerator)

	if rec.Code != nethttp.StatusCreated {
		t.Fatalf("status = %d, want 201 (body %s)", rec.Code, rec.Body.String())
	}
	got := decodeData(t, env)
	want := map[string]any{
		"id":         "b-1",
		"user_id":    "u-2",
		"report_id":  "11111111-1111-4111-8111-111111111111",
		"reason":     "spam",
		"banned_by":  "mod-1",
		"expires_at": "2026-11-20T12:00:00Z",
		"created_at": "2026-10-29T10:00:00Z",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("data[%q] = %v, want %v", k, got[k], v)
		}
	}
}

func TestModerationHandler_BanUser_NoExpiry_IsPermanent(t *testing.T) {
	svc := &fakeBanService{banFn: func(_ context.Context, in ban.Input) (ban.Ban, error) {
		if in.ExpiresAt != nil {
			t.Errorf("ExpiresAt = %v, want nil", in.ExpiresAt)
		}
		return ban.Ban{ID: "b-1", UserID: in.UserID, Reason: in.Reason, CreatedAt: createdAt}, nil
	}}

	rec, env := serveModeration(t, svc, nethttp.MethodPost, "/api/v1/moderation/users/u-2/ban",
		`{"report_id":"11111111-1111-4111-8111-111111111111","reason":"spam"}`, "admin-1", user.RoleAdmin)

	if rec.Code != nethttp.StatusCreated {
		t.Fatalf("status = %d, want 201 (body %s)", rec.Code, rec.Body.String())
	}
	if got := decodeData(t, env); got["expires_at"] != nil {
		t.Errorf("expires_at = %v, want null", got["expires_at"])
	}
}

func TestModerationHandler_Guards_RejectUnauthorizedCallers(t *testing.T) {
	svc := &fakeBanService{banFn: func(context.Context, ban.Input) (ban.Ban, error) {
		t.Error("Ban() must not be called")
		return ban.Ban{}, nil
	}}

	rec, env := serveModeration(t, svc, nethttp.MethodPost, "/api/v1/moderation/users/u-2/ban", `{"report_id":"11111111-1111-4111-8111-111111111111","reason":"spam"}`, "", "")
	assertError(t, rec, env, nethttp.StatusUnauthorized, "UNAUTHORIZED", "")

	rec, env = serveModeration(t, svc, nethttp.MethodPost, "/api/v1/moderation/users/u-2/ban", `{"report_id":"11111111-1111-4111-8111-111111111111","reason":"spam"}`, "u-1", user.RoleUser)
	assertError(t, rec, env, nethttp.StatusForbidden, "FORBIDDEN", "insufficient permissions")
}

func TestModerationHandler_BanUser_InvalidPayload_Returns400(t *testing.T) {
	svc := &fakeBanService{banFn: func(context.Context, ban.Input) (ban.Ban, error) {
		t.Error("Ban() must not be called")
		return ban.Ban{}, nil
	}}

	for _, body := range []string{
		`{"report_id":"11111111-1111-4111-8111-111111111111"}`,
		`{"reason":"spam"}`,
		`{"report_id":"not-a-uuid","reason":"spam"}`,
	} {
		rec, env := serveModeration(t, svc, nethttp.MethodPost, "/api/v1/moderation/users/u-2/ban", body, "mod-1", user.RoleModerator)
		assertError(t, rec, env, nethttp.StatusBadRequest, "VALIDATION_ERROR", "")
	}
}

func TestModerationHandler_BannedModerator_Returns403AccountBanned(t *testing.T) {
	svc := &fakeBanService{banFn: func(context.Context, ban.Input) (ban.Ban, error) {
		t.Error("Ban() must not be called")
		return ban.Ban{}, nil
	}}
	srv, err := NewServer(Deps{
		Users:            NewUserHandler(&fakeUserService{}),
		Auth:             noopAuthRoutes{},
		Moderation:       NewModerationHandler(svc),
		Authenticate:     testAuthenticateWithRole,
		RejectBanned:     middleware.RejectBanned(bannedUsers{"mod-1": true}),
		RequireModerator: middleware.RequireRole(user.RoleModerator, user.RoleAdmin),
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	req := httptest.NewRequest(nethttp.MethodPost, "/api/v1/moderation/users/u-2/ban", strings.NewReader(`{"report_id":"11111111-1111-4111-8111-111111111111","reason":"spam"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(testUserHeader, "mod-1")
	req.Header.Set(testRoleHeader, string(user.RoleModerator))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	var env envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON body %q: %v", rec.Body.String(), err)
	}
	assertError(t, rec, env, nethttp.StatusForbidden, "ACCOUNT_BANNED", "account permanently banned")
}

// bannedUsers is a middleware.UserBanChecker banning the listed users.
type bannedUsers map[string]bool

func (b bannedUsers) CheckUser(_ context.Context, userID string) error {
	if b[userID] {
		return &apperror.Error{Code: apperror.CodeAccountBanned, Message: "account permanently banned"}
	}
	return nil
}

func TestModerationHandler_BanUser_ServiceError_IsMapped(t *testing.T) {
	svc := &fakeBanService{banFn: func(context.Context, ban.Input) (ban.Ban, error) {
		return ban.Ban{}, ban.ErrCannotBanStaff
	}}

	rec, env := serveModeration(t, svc, nethttp.MethodPost, "/api/v1/moderation/users/u-2/ban", `{"report_id":"11111111-1111-4111-8111-111111111111","reason":"spam"}`, "mod-1", user.RoleModerator)

	assertError(t, rec, env, nethttp.StatusForbidden, "FORBIDDEN", "only an admin can ban a moderator or an admin")
}

func TestModerationHandler_GetBan_NotBanned_Returns404(t *testing.T) {
	svc := &fakeBanService{activeFn: func(_ context.Context, userID string) (ban.Ban, error) {
		if userID != "u-2" {
			t.Errorf("userID = %q, want u-2", userID)
		}
		return ban.Ban{}, ban.ErrNotBanned
	}}

	rec, env := serveModeration(t, svc, nethttp.MethodGet, "/api/v1/moderation/users/u-2/ban", "", "mod-1", user.RoleModerator)

	assertError(t, rec, env, nethttp.StatusNotFound, "NOT_FOUND", "user is not banned")
}

func TestModerationHandler_LiftBan_Returns204(t *testing.T) {
	called := false
	svc := &fakeBanService{liftFn: func(_ context.Context, userID string, by ban.Moderator) error {
		called = true
		if userID != "u-2" || by.ID != "mod-1" {
			t.Errorf("Lift(%q, %+v)", userID, by)
		}
		return nil
	}}

	rec, _ := serveModeration(t, svc, nethttp.MethodDelete, "/api/v1/moderation/users/u-2/ban", "", "mod-1", user.RoleModerator)

	if rec.Code != nethttp.StatusNoContent || !called {
		t.Fatalf("status = %d, called = %v, want 204 and a call", rec.Code, called)
	}
}

func TestNewServer_ModerationWithoutGuard_Fails(t *testing.T) {
	_, err := NewServer(Deps{
		Users:        NewUserHandler(&fakeUserService{}),
		Auth:         noopAuthRoutes{},
		Moderation:   NewModerationHandler(&fakeBanService{}),
		Authenticate: testAuthenticate,
	})
	if err == nil {
		t.Fatal("NewServer() error = nil, want an error when RequireModerator is missing")
	}
}

func TestServer_BannedUser_CanOnlyDeleteTheirAccount(t *testing.T) {
	deleted := ""
	users := &fakeUserService{
		getFn: func(context.Context, string) (user.User, error) {
			t.Error("Get() must not be called for a banned user")
			return user.User{}, nil
		},
		deleteFn: func(_ context.Context, id string) error {
			deleted = id
			return nil
		},
	}
	srv, err := NewServer(Deps{
		Users:        NewUserHandler(users),
		Auth:         noopAuthRoutes{},
		Authenticate: testAuthenticateWithRole,
		RejectBanned: middleware.RejectBanned(bannedUsers{"u-1": true}),
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	do := func(method, path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set(testUserHeader, "u-1")
		req.Header.Set(testRoleHeader, string(user.RoleUser))
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		return rec
	}

	for _, path := range []string{"/api/v1/users/me", "/api/v1/users/u-2"} {
		if rec := do(nethttp.MethodGet, path); rec.Code != nethttp.StatusForbidden {
			t.Errorf("GET %s status = %d, want 403", path, rec.Code)
		}
	}
	if rec := do(nethttp.MethodDelete, "/api/v1/users/me"); rec.Code != nethttp.StatusNoContent || deleted != "u-1" {
		t.Errorf("DELETE /users/me status = %d, deleted = %q, want 204 and u-1", rec.Code, deleted)
	}
}
