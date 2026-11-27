//go:build integration

package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/adapters/http"
	"github.com/traily-org/server/internal/adapters/http/middleware"
	"github.com/traily-org/server/internal/auth"
	"github.com/traily-org/server/internal/domain/ban"
	"github.com/traily-org/server/internal/domain/user"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/internal/testutil"
)

// banFlow is the real server wired as in cmd/main.go, on the shared test
// database.
type banFlow struct {
	h    nethttp.Handler
	pool *pgxpool.Pool
}

func newBanFlow(t *testing.T) banFlow {
	t.Helper()
	gin.SetMode(gin.TestMode)

	pool := testutil.NewPool(t)
	users := user.NewService(postgres.NewUserRepository(pool))
	authService := auth.NewService(users, postgres.NewRefreshTokenRepository(pool), auth.Config{
		AccessSecret:  []byte(strings.Repeat("a", 32)),
		RefreshSecret: []byte(strings.Repeat("r", 32)),
		AccessTTL:     15 * time.Minute,
		RefreshTTL:    30 * 24 * time.Hour,
		Issuer:        "traily-test",
	})
	users.SetSessionRevoker(authService)

	bans := ban.NewService(postgres.NewBanRepository(pool), users)
	bans.SetSessionRevoker(authService)
	authService.SetBanChecker(bans)

	srv, err := http.NewServer(http.Deps{
		Users:            http.NewUserHandler(users),
		Auth:             http.NewAuthHandler(authService),
		Moderation:       http.NewModerationHandler(bans),
		Authenticate:     middleware.Authenticate(authService),
		RejectBanned:     middleware.RejectBanned(bans),
		RequireModerator: middleware.RequireRole(user.RoleModerator, user.RoleAdmin),
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	return banFlow{h: srv.Handler(), pool: pool}
}

// banResult is a decoded API response.
type banResult struct {
	status  int
	data    map[string]any
	code    string
	message string
}

func (f banFlow) do(t *testing.T, method, path, token, body string) banResult {
	t.Helper()

	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	f.h.ServeHTTP(rec, req)

	res := banResult{status: rec.Code}
	if rec.Body.Len() == 0 {
		return res
	}
	var env struct {
		Data  map[string]any `json:"data"`
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("%s %s: decode body %q: %v", method, path, rec.Body.String(), err)
	}
	res.data = env.Data
	if env.Error != nil {
		res.code, res.message = env.Error.Code, env.Error.Message
	}
	return res
}

// expect fails the test when the response does not have the given status and,
// when code is not empty, the given error code.
func (r banResult) expect(t *testing.T, step string, status int, code string) {
	t.Helper()
	if r.status != status || r.code != code {
		t.Fatalf("%s: got %d %q (%s), want %d %q", step, r.status, r.code, r.message, status, code)
	}
}

// tokens logs in with the seed password and returns the access and refresh
// tokens.
func (f banFlow) tokens(t *testing.T, email string) (string, string) {
	t.Helper()

	res := f.do(t, nethttp.MethodPost, "/api/v1/auth/login", "",
		fmt.Sprintf(`{"email":%q,"password":%q}`, email, testutil.SeedPassword))
	res.expect(t, "login "+email, nethttp.StatusOK, "")
	tok, _ := res.data["tokens"].(map[string]any)
	access, _ := tok["access_token"].(string)
	refresh, _ := tok["refresh_token"].(string)
	if access == "" || refresh == "" {
		t.Fatalf("login %s: missing tokens in %v", email, res.data)
	}
	return access, refresh
}

func (f banFlow) login(t *testing.T, email string) banResult {
	t.Helper()
	return f.do(t, nethttp.MethodPost, "/api/v1/auth/login", "",
		fmt.Sprintf(`{"email":%q,"password":%q}`, email, testutil.SeedPassword))
}

func banPath(userID string) string {
	return "/api/v1/moderation/users/" + userID + "/ban"
}

func banBody(reportID, expiresAt string) string {
	if expiresAt == "" {
		return fmt.Sprintf(`{"report_id":%q,"reason":"harassment"}`, reportID)
	}
	return fmt.Sprintf(`{"report_id":%q,"reason":"harassment","expires_at":%q}`, reportID, expiresAt)
}

func (f banFlow) scalar(t *testing.T, query string, args ...any) string {
	t.Helper()

	var v string
	if err := f.pool.QueryRow(context.Background(), query, args...).Scan(&v); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return v
}

func TestBanFlow_TemporaryBan_LocksTheUserOutEverywhere(t *testing.T) {
	f := newBanFlow(t)
	mod := testutil.SeedAccount(t, f.pool, "mod@example.com", user.RoleModerator)
	jane := testutil.SeedAccount(t, f.pool, "jane.doe@gmail.com", user.RoleUser)
	reportID := testutil.SeedReport(t, f.pool, mod.ID, "user", jane.ID)

	janeAccess, janeRefresh := f.tokens(t, jane.Email)
	modAccess, _ := f.tokens(t, mod.Email)
	f.do(t, nethttp.MethodGet, "/api/v1/users/me", janeAccess, "").expect(t, "me before the ban", nethttp.StatusOK, "")

	expiresAt := time.Now().Add(7 * 24 * time.Hour).UTC().Truncate(time.Second)
	res := f.do(t, nethttp.MethodPost, banPath(jane.ID), modAccess, banBody(reportID, expiresAt.Format(time.RFC3339)))
	res.expect(t, "ban", nethttp.StatusCreated, "")
	if res.data["report_id"] != reportID || res.data["expires_at"] != expiresAt.Format(time.RFC3339) {
		t.Fatalf("ban: data = %v, want report_id %s and expires_at %s", res.data, reportID, expiresAt.Format(time.RFC3339))
	}

	wantMessage := "account banned until " + expiresAt.Format(time.RFC3339)
	me := f.do(t, nethttp.MethodGet, "/api/v1/users/me", janeAccess, "")
	me.expect(t, "me with the access token issued before the ban", nethttp.StatusForbidden, "ACCOUNT_BANNED")
	if me.message != wantMessage {
		t.Errorf("me: message = %q, want %q", me.message, wantMessage)
	}
	f.do(t, nethttp.MethodGet, "/api/v1/users/"+mod.ID, janeAccess, "").
		expect(t, "public profile while banned", nethttp.StatusForbidden, "ACCOUNT_BANNED")
	f.do(t, nethttp.MethodPost, "/api/v1/auth/refresh", "", fmt.Sprintf(`{"refresh_token":%q}`, janeRefresh)).
		expect(t, "refresh with a token revoked by the ban", nethttp.StatusUnauthorized, "UNAUTHORIZED")
	f.login(t, jane.Email).expect(t, "login while banned", nethttp.StatusForbidden, "ACCOUNT_BANNED")
	f.do(t, nethttp.MethodPost, "/api/v1/auth/login", "", fmt.Sprintf(`{"email":%q,"password":"wrong-password"}`, jane.Email)).
		expect(t, "login with a wrong password does not reveal the ban", nethttp.StatusUnauthorized, "UNAUTHORIZED")

	if got := f.scalar(t, `SELECT status::text || ' ' || (reviewed_by = $2)::text FROM reports WHERE id = $1`, reportID, mod.ID); got != "reviewed true" {
		t.Errorf("report = %q, want reviewed by the moderator", got)
	}
	if got := f.scalar(t, `SELECT count(*)::text FROM moderation_logs WHERE action = 'ban' AND target_id = $1 AND report_id = $2`, jane.ID, reportID); got != "1" {
		t.Errorf("ban logs = %s, want 1", got)
	}
}

func TestBanFlow_BannedUserDeletesAccount_CannotSignUpAgain(t *testing.T) {
	f := newBanFlow(t)
	mod := testutil.SeedAccount(t, f.pool, "mod@example.com", user.RoleModerator)
	jane := testutil.SeedAccount(t, f.pool, "jane.doe@gmail.com", user.RoleUser)
	reportID := testutil.SeedReport(t, f.pool, mod.ID, "user", jane.ID)
	janeAccess, _ := f.tokens(t, jane.Email)
	modAccess, _ := f.tokens(t, mod.Email)

	f.do(t, nethttp.MethodPost, banPath(jane.ID), modAccess, banBody(reportID, "")).expect(t, "ban", nethttp.StatusCreated, "")

	f.do(t, nethttp.MethodDelete, "/api/v1/users/me", janeAccess, "").
		expect(t, "delete own account while banned", nethttp.StatusNoContent, "")
	if got := f.scalar(t, `SELECT count(*)::text FROM users WHERE id = $1`, jane.ID); got != "0" {
		t.Fatalf("users with the deleted id = %s, want 0", got)
	}

	for _, email := range []string{"jane.doe@gmail.com", "JaneDoe+again@googlemail.com", "j.a.n.e.d.o.e@gmail.com"} {
		f.do(t, nethttp.MethodPost, "/api/v1/auth/register", "",
			fmt.Sprintf(`{"email":%q,"password":"supersecret","name":"Jane"}`, email)).
			expect(t, "register again with "+email, nethttp.StatusForbidden, "ACCOUNT_BANNED")
	}
	f.do(t, nethttp.MethodPost, "/api/v1/auth/register", "",
		`{"email":"someone.else@gmail.com","password":"supersecret","name":"Other"}`).
		expect(t, "register with another address", nethttp.StatusCreated, "")
}

func TestBanFlow_LiftBan_RestoresAccess(t *testing.T) {
	f := newBanFlow(t)
	mod := testutil.SeedAccount(t, f.pool, "mod@example.com", user.RoleModerator)
	admin := testutil.SeedAccount(t, f.pool, "admin@example.com", user.RoleAdmin)
	jane := testutil.SeedAccount(t, f.pool, "jane@example.com", user.RoleUser)
	reportID := testutil.SeedReport(t, f.pool, mod.ID, "user", jane.ID)
	modAccess, _ := f.tokens(t, mod.Email)
	adminAccess, _ := f.tokens(t, admin.Email)

	f.do(t, nethttp.MethodPost, banPath(jane.ID), modAccess, banBody(reportID, "")).expect(t, "permanent ban", nethttp.StatusCreated, "")

	got := f.do(t, nethttp.MethodGet, banPath(jane.ID), modAccess, "")
	got.expect(t, "get ban", nethttp.StatusOK, "")
	if got.data["expires_at"] != nil || got.data["report_id"] != reportID {
		t.Errorf("get ban: data = %v, want a permanent ban from %s", got.data, reportID)
	}
	f.login(t, jane.Email).expect(t, "login while permanently banned", nethttp.StatusForbidden, "ACCOUNT_BANNED")

	f.do(t, nethttp.MethodDelete, banPath(jane.ID), adminAccess, "").expect(t, "lift", nethttp.StatusNoContent, "")

	f.login(t, jane.Email).expect(t, "login after lift", nethttp.StatusOK, "")
	f.do(t, nethttp.MethodGet, banPath(jane.ID), modAccess, "").expect(t, "get ban after lift", nethttp.StatusNotFound, "NOT_FOUND")
	f.do(t, nethttp.MethodDelete, banPath(jane.ID), modAccess, "").expect(t, "lift twice", nethttp.StatusNotFound, "NOT_FOUND")
	if got := f.scalar(t, `SELECT count(*)::text FROM banned_identifiers`); got != "0" {
		t.Errorf("banned identifiers after lift = %s, want 0", got)
	}
	if got := f.scalar(t, `SELECT count(*)::text FROM moderation_logs WHERE action = 'unban' AND moderator_id = $1`, admin.ID); got != "1" {
		t.Errorf("unban logs = %s, want 1", got)
	}
}

func TestBanFlow_ExpiredBan_DoesNotBlock(t *testing.T) {
	f := newBanFlow(t)
	jane := testutil.SeedAccount(t, f.pool, "jane@example.com", user.RoleUser)
	testutil.SeedExpiredBan(t, f.pool, jane.ID)

	access, _ := f.tokens(t, jane.Email)
	f.do(t, nethttp.MethodGet, "/api/v1/users/me", access, "").expect(t, "me after the ban ended", nethttp.StatusOK, "")
}

func TestBanFlow_BanOnReportedContent_BansItsAuthor(t *testing.T) {
	f := newBanFlow(t)
	mod := testutil.SeedAccount(t, f.pool, "mod@example.com", user.RoleModerator)
	jane := testutil.SeedAccount(t, f.pool, "jane@example.com", user.RoleUser)
	bob := testutil.SeedAccount(t, f.pool, "bob@example.com", user.RoleUser)
	reportID := testutil.SeedReport(t, f.pool, mod.ID, "photo", testutil.SeedPhoto(t, f.pool, jane.ID))
	modAccess, _ := f.tokens(t, mod.Email)

	f.do(t, nethttp.MethodPost, banPath(bob.ID), modAccess, banBody(reportID, "")).
		expect(t, "ban someone else than the author", nethttp.StatusBadRequest, "VALIDATION_ERROR")
	f.do(t, nethttp.MethodPost, banPath(jane.ID), modAccess, banBody(reportID, "")).
		expect(t, "ban the author of the photo", nethttp.StatusCreated, "")
	f.login(t, jane.Email).expect(t, "login of the author", nethttp.StatusForbidden, "ACCOUNT_BANNED")
	f.login(t, bob.Email).expect(t, "login of the other user", nethttp.StatusOK, "")
}

func TestBanFlow_Guards_RefuseInvalidBans(t *testing.T) {
	f := newBanFlow(t)
	mod := testutil.SeedAccount(t, f.pool, "mod@example.com", user.RoleModerator)
	otherMod := testutil.SeedAccount(t, f.pool, "mod2@example.com", user.RoleModerator)
	jane := testutil.SeedAccount(t, f.pool, "jane@example.com", user.RoleUser)
	bob := testutil.SeedAccount(t, f.pool, "bob@example.com", user.RoleUser)
	reportJane := testutil.SeedReport(t, f.pool, bob.ID, "user", jane.ID)
	reportMod := testutil.SeedReport(t, f.pool, bob.ID, "user", otherMod.ID)
	dismissed := testutil.SeedReport(t, f.pool, mod.ID, "user", jane.ID)
	if _, err := f.pool.Exec(context.Background(), `UPDATE reports SET status = 'dismissed' WHERE id = $1`, dismissed); err != nil {
		t.Fatalf("dismiss report: %v", err)
	}
	modAccess, _ := f.tokens(t, mod.Email)
	bobAccess, _ := f.tokens(t, bob.Email)
	past := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)

	tests := []struct {
		name   string
		token  string
		target string
		body   string
		status int
		code   string
	}{
		{"no token", "", jane.ID, banBody(reportJane, ""), nethttp.StatusUnauthorized, "UNAUTHORIZED"},
		{"plain user", bobAccess, jane.ID, banBody(reportJane, ""), nethttp.StatusForbidden, "FORBIDDEN"},
		{"no report", modAccess, jane.ID, `{"reason":"harassment"}`, nethttp.StatusBadRequest, "VALIDATION_ERROR"},
		{"unknown report", modAccess, jane.ID, banBody("00000000-0000-4000-8000-000000000000", ""), nethttp.StatusNotFound, "NOT_FOUND"},
		{"report about another user", modAccess, bob.ID, banBody(reportJane, ""), nethttp.StatusBadRequest, "VALIDATION_ERROR"},
		{"dismissed report", modAccess, jane.ID, banBody(dismissed, ""), nethttp.StatusConflict, "CONFLICT"},
		{"end date in the past", modAccess, jane.ID, banBody(reportJane, past), nethttp.StatusBadRequest, "VALIDATION_ERROR"},
		{"moderator bans a moderator", modAccess, otherMod.ID, banBody(reportMod, ""), nethttp.StatusForbidden, "FORBIDDEN"},
		{"self ban", modAccess, mod.ID, banBody(reportJane, ""), nethttp.StatusBadRequest, "VALIDATION_ERROR"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f.do(t, nethttp.MethodPost, banPath(tt.target), tt.token, tt.body).expect(t, tt.name, tt.status, tt.code)
		})
	}

	if got := f.scalar(t, `SELECT count(*)::text FROM user_bans`); got != "0" {
		t.Errorf("bans after refused requests = %s, want 0", got)
	}
	f.login(t, jane.Email).expect(t, "login of the unbanned user", nethttp.StatusOK, "")
}

func TestBanFlow_BannedModerator_LosesModerationAccess(t *testing.T) {
	f := newBanFlow(t)
	admin := testutil.SeedAccount(t, f.pool, "admin@example.com", user.RoleAdmin)
	mod := testutil.SeedAccount(t, f.pool, "mod@example.com", user.RoleModerator)
	jane := testutil.SeedAccount(t, f.pool, "jane@example.com", user.RoleUser)
	reportMod := testutil.SeedReport(t, f.pool, jane.ID, "user", mod.ID)
	reportJane := testutil.SeedReport(t, f.pool, admin.ID, "user", jane.ID)
	adminAccess, _ := f.tokens(t, admin.Email)
	modAccess, _ := f.tokens(t, mod.Email)

	f.do(t, nethttp.MethodPost, banPath(mod.ID), adminAccess, banBody(reportMod, "")).
		expect(t, "admin bans a moderator", nethttp.StatusCreated, "")

	f.do(t, nethttp.MethodPost, banPath(jane.ID), modAccess, banBody(reportJane, "")).
		expect(t, "banned moderator tries to ban", nethttp.StatusForbidden, "ACCOUNT_BANNED")
	if got := f.scalar(t, `SELECT count(*)::text FROM user_bans WHERE user_id = $1`, jane.ID); got != "0" {
		t.Errorf("bans of jane = %s, want 0", got)
	}
}
