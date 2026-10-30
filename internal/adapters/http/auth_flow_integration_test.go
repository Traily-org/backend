//go:build integration

package http_test

import (
	"bytes"
	"encoding/json"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/adapters/http"
	"github.com/traily-org/server/internal/adapters/http/middleware"
	"github.com/traily-org/server/internal/auth"
	"github.com/traily-org/server/internal/domain/user"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/internal/testutil"
)

type flowTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type flowEnvelope struct {
	Data struct {
		User   map[string]any `json:"user"`
		Tokens flowTokens     `json:"tokens"`
		Email  string         `json:"email"`
	} `json:"data"`
	Error *struct {
		Code string `json:"code"`
	} `json:"error"`
}

// newFlowServer wires the real repositories, services, middlewares and router
// on the shared test database.
func newFlowServer(t *testing.T, rateLimit gin.HandlerFunc) nethttp.Handler {
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

	srv, err := http.NewServer(http.Deps{
		Users:         http.NewUserHandler(users),
		Auth:          http.NewAuthHandler(authService),
		Authenticate:  middleware.Authenticate(authService),
		AuthRateLimit: rateLimit,
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	return srv.Handler()
}

func call(t *testing.T, h nethttp.Handler, method, path, token, body string) (int, flowEnvelope) {
	t.Helper()

	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	var env flowEnvelope
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
			t.Fatalf("%s %s: decode body %q: %v", method, path, rec.Body.String(), err)
		}
	}
	return rec.Code, env
}

func expectStatus(t *testing.T, step string, got, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("%s: status = %d, want %d", step, got, want)
	}
}

func TestAuthFlow_RegisterMeRefreshLogout_BehavesEndToEnd(t *testing.T) {
	h := newFlowServer(t, func(c *gin.Context) { c.Next() })

	status, reg := call(t, h, nethttp.MethodPost, "/api/v1/auth/register", "",
		`{"email":"Hiker@Example.com","password":"supersecret","name":"Hiker"}`)
	expectStatus(t, "register", status, nethttp.StatusCreated)
	if reg.Data.Tokens.AccessToken == "" || reg.Data.Tokens.RefreshToken == "" {
		t.Fatalf("register: missing tokens in %+v", reg.Data)
	}

	status, me := call(t, h, nethttp.MethodGet, "/api/v1/users/me", reg.Data.Tokens.AccessToken, "")
	expectStatus(t, "me", status, nethttp.StatusOK)
	if me.Data.Email != "hiker@example.com" {
		t.Fatalf("me: email = %q, want the normalized email", me.Data.Email)
	}

	status, anon := call(t, h, nethttp.MethodGet, "/api/v1/users/me", "", "")
	expectStatus(t, "me without token", status, nethttp.StatusUnauthorized)
	if anon.Error == nil || anon.Error.Code != "UNAUTHORIZED" {
		t.Fatalf("me without token: error = %+v, want UNAUTHORIZED", anon.Error)
	}

	status, _ = call(t, h, nethttp.MethodPost, "/api/v1/auth/login", "",
		`{"email":"hiker@example.com","password":"wrong-password"}`)
	expectStatus(t, "login with wrong password", status, nethttp.StatusUnauthorized)

	status, login := call(t, h, nethttp.MethodPost, "/api/v1/auth/login", "",
		`{"email":"HIKER@example.com","password":"supersecret"}`)
	expectStatus(t, "login", status, nethttp.StatusOK)

	refreshBody := func(token string) string { return `{"refresh_token":"` + token + `"}` }

	status, rotated := call(t, h, nethttp.MethodPost, "/api/v1/auth/refresh", "", refreshBody(reg.Data.Tokens.RefreshToken))
	expectStatus(t, "refresh", status, nethttp.StatusOK)
	if rotated.Data.Tokens.RefreshToken == reg.Data.Tokens.RefreshToken {
		t.Fatal("refresh: refresh token was not rotated")
	}

	// Reusing a rotated token is treated as theft: every session is revoked.
	status, _ = call(t, h, nethttp.MethodPost, "/api/v1/auth/refresh", "", refreshBody(reg.Data.Tokens.RefreshToken))
	expectStatus(t, "reuse of rotated token", status, nethttp.StatusUnauthorized)
	status, _ = call(t, h, nethttp.MethodPost, "/api/v1/auth/refresh", "", refreshBody(rotated.Data.Tokens.RefreshToken))
	expectStatus(t, "refresh after reuse detection", status, nethttp.StatusUnauthorized)
	status, _ = call(t, h, nethttp.MethodPost, "/api/v1/auth/refresh", "", refreshBody(login.Data.Tokens.RefreshToken))
	expectStatus(t, "other session after reuse detection", status, nethttp.StatusUnauthorized)

	status, relog := call(t, h, nethttp.MethodPost, "/api/v1/auth/login", "",
		`{"email":"hiker@example.com","password":"supersecret"}`)
	expectStatus(t, "login again", status, nethttp.StatusOK)

	status, _ = call(t, h, nethttp.MethodPost, "/api/v1/auth/logout", "", refreshBody(relog.Data.Tokens.RefreshToken))
	expectStatus(t, "logout", status, nethttp.StatusNoContent)
	status, _ = call(t, h, nethttp.MethodPost, "/api/v1/auth/refresh", "", refreshBody(relog.Data.Tokens.RefreshToken))
	expectStatus(t, "refresh after logout", status, nethttp.StatusUnauthorized)
}

func TestAuthFlow_DuplicateRegister_Returns409(t *testing.T) {
	h := newFlowServer(t, func(c *gin.Context) { c.Next() })
	body := `{"email":"dup@example.com","password":"supersecret","name":"Dup"}`

	status, _ := call(t, h, nethttp.MethodPost, "/api/v1/auth/register", "", body)
	expectStatus(t, "first register", status, nethttp.StatusCreated)

	status, env := call(t, h, nethttp.MethodPost, "/api/v1/auth/register", "",
		`{"email":"DUP@example.com","password":"supersecret","name":"Dup"}`)
	expectStatus(t, "second register", status, nethttp.StatusConflict)
	if env.Error == nil || env.Error.Code != "CONFLICT" {
		t.Fatalf("second register: error = %+v, want CONFLICT", env.Error)
	}
}

func TestAuthFlow_TooManyLogins_Returns429(t *testing.T) {
	h := newFlowServer(t, middleware.RateLimit(2, time.Minute))
	body := `{"email":"nobody@example.com","password":"whatever-pass"}`

	for i := 0; i < 2; i++ {
		status, _ := call(t, h, nethttp.MethodPost, "/api/v1/auth/login", "", body)
		expectStatus(t, "login under the limit", status, nethttp.StatusUnauthorized)
	}

	status, env := call(t, h, nethttp.MethodPost, "/api/v1/auth/login", "", body)
	expectStatus(t, "login over the limit", status, nethttp.StatusTooManyRequests)
	if env.Error == nil || env.Error.Code != "RATE_LIMITED" {
		t.Fatalf("login over the limit: error = %+v, want RATE_LIMITED", env.Error)
	}
}

func TestAuthFlow_LoginRateLimited_RefreshStillAllowed(t *testing.T) {
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
	srv, err := http.NewServer(http.Deps{
		Users:         http.NewUserHandler(users),
		Auth:          http.NewAuthHandler(authService),
		Authenticate:  middleware.Authenticate(authService),
		AuthRateLimit: middleware.RateLimit(1, time.Minute),
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	h := srv.Handler()

	status, reg := call(t, h, nethttp.MethodPost, "/api/v1/auth/register", "",
		`{"email":"limited@example.com","password":"supersecret","name":"Limited"}`)
	expectStatus(t, "register", status, nethttp.StatusCreated)
	status, _ = call(t, h, nethttp.MethodPost, "/api/v1/auth/login", "",
		`{"email":"limited@example.com","password":"supersecret"}`)
	expectStatus(t, "login over the limit", status, nethttp.StatusTooManyRequests)

	status, _ = call(t, h, nethttp.MethodPost, "/api/v1/auth/refresh", "",
		`{"refresh_token":"`+reg.Data.Tokens.RefreshToken+`"}`)
	expectStatus(t, "refresh with its own budget", status, nethttp.StatusOK)
}

func TestAuthFlow_PasswordChange_RequiresCurrentPasswordAndRevokesSessions(t *testing.T) {
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
	srv, err := http.NewServer(http.Deps{
		Users:        http.NewUserHandler(users),
		Auth:         http.NewAuthHandler(authService),
		Authenticate: middleware.Authenticate(authService),
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	h := srv.Handler()

	status, reg := call(t, h, nethttp.MethodPost, "/api/v1/auth/register", "",
		`{"email":"owner@example.com","password":"supersecret","name":"Owner"}`)
	expectStatus(t, "register", status, nethttp.StatusCreated)
	access := reg.Data.Tokens.AccessToken

	status, env := call(t, h, nethttp.MethodPut, "/api/v1/users/me", access,
		`{"email":"attacker@example.com","name":"Owner"}`)
	expectStatus(t, "email change without current password", status, nethttp.StatusBadRequest)
	if env.Error == nil || env.Error.Code != "VALIDATION_ERROR" {
		t.Fatalf("email change without current password: error = %+v", env.Error)
	}

	status, _ = call(t, h, nethttp.MethodPut, "/api/v1/users/me", access,
		`{"email":"owner@example.com","name":"Owner","password":"hijacked-pass","current_password":"guess-guess"}`)
	expectStatus(t, "password change with wrong current password", status, nethttp.StatusForbidden)

	status, _ = call(t, h, nethttp.MethodPut, "/api/v1/users/me", access,
		`{"email":"owner@example.com","name":"Owner","password":"new-password","current_password":"supersecret"}`)
	expectStatus(t, "password change", status, nethttp.StatusOK)

	status, _ = call(t, h, nethttp.MethodPost, "/api/v1/auth/refresh", "",
		`{"refresh_token":"`+reg.Data.Tokens.RefreshToken+`"}`)
	expectStatus(t, "refresh after password change", status, nethttp.StatusUnauthorized)

	status, _ = call(t, h, nethttp.MethodPost, "/api/v1/auth/login", "",
		`{"email":"owner@example.com","password":"new-password"}`)
	expectStatus(t, "login with new password", status, nethttp.StatusOK)
}
