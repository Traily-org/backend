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

	"github.com/traily-org/server/internal/auth"
	"github.com/traily-org/server/internal/domain/user"
	"github.com/traily-org/server/internal/pkg/apperror"
)

type fakeAuthService struct {
	registerFn func(ctx context.Context, u user.User) (user.User, auth.TokenPair, error)
	loginFn    func(ctx context.Context, email, password string) (user.User, auth.TokenPair, error)
	refreshFn  func(ctx context.Context, refreshToken string) (auth.TokenPair, error)
	logoutFn   func(ctx context.Context, refreshToken string) error
}

func (f *fakeAuthService) Register(ctx context.Context, u user.User) (user.User, auth.TokenPair, error) {
	return f.registerFn(ctx, u)
}

func (f *fakeAuthService) Login(ctx context.Context, email, password string) (user.User, auth.TokenPair, error) {
	return f.loginFn(ctx, email, password)
}

func (f *fakeAuthService) Refresh(ctx context.Context, refreshToken string) (auth.TokenPair, error) {
	return f.refreshFn(ctx, refreshToken)
}

func (f *fakeAuthService) Logout(ctx context.Context, refreshToken string) error {
	return f.logoutFn(ctx, refreshToken)
}

type authEnvelope struct {
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

var (
	authZone      = time.FixedZone("CEST", 2*60*60)
	authCreatedAt = time.Date(2026, 10, 29, 12, 0, 0, 0, authZone)
	authUser      = user.User{ID: "u-1", Email: "alice@example.com", Password: "hashed-secret", Name: "Alice", Role: user.RoleUser, CreatedAt: authCreatedAt, UpdatedAt: authCreatedAt}
	authTokens    = auth.TokenPair{
		AccessToken:      "access.jwt",
		AccessExpiresAt:  time.Date(2026, 10, 29, 12, 15, 0, 0, authZone),
		RefreshToken:     "refresh-opaque",
		RefreshExpiresAt: time.Date(2026, 11, 28, 12, 0, 0, 0, authZone),
	}
	wantTokensJSON = map[string]any{
		"access_token":       "access.jwt",
		"access_expires_at":  "2026-10-29T10:15:00Z",
		"refresh_token":      "refresh-opaque",
		"refresh_expires_at": "2026-11-28T10:00:00Z",
		"token_type":         "Bearer",
	}
)

func serveAuth(t *testing.T, svc *fakeAuthService, path, body string) (*httptest.ResponseRecorder, authEnvelope) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	h := NewAuthHandler(svc)
	r := gin.New()
	g := r.Group("/api/v1/auth")
	g.POST("/register", h.Register)
	g.POST("/login", h.Login)
	g.POST("/refresh", h.Refresh)
	g.POST("/logout", h.Logout)

	req := httptest.NewRequest(nethttp.MethodPost, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	var env authEnvelope
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
			t.Fatalf("invalid JSON body %q: %v", rec.Body.String(), err)
		}
	}
	return rec, env
}

func assertAuthError(t *testing.T, rec *httptest.ResponseRecorder, env authEnvelope, status int, code string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d (body %s)", rec.Code, status, rec.Body.String())
	}
	if env.Error == nil || env.Error.Code != code {
		t.Fatalf("error = %+v, want code %s", env.Error, code)
	}
	if env.Data != nil {
		t.Fatalf("error response carries data: %s", env.Data)
	}
}

func decodeAuthData(t *testing.T, env authEnvelope) map[string]any {
	t.Helper()
	var data map[string]any
	if err := json.Unmarshal(env.Data, &data); err != nil {
		t.Fatalf("invalid data %s: %v", env.Data, err)
	}
	return data
}

func assertJSONEqual(t *testing.T, name string, got, want any) {
	t.Helper()
	g, _ := json.Marshal(got)
	w, _ := json.Marshal(want)
	if string(g) != string(w) {
		t.Fatalf("%s = %s, want %s", name, g, w)
	}
}

func notCalled(t *testing.T) *fakeAuthService {
	fail := func() { t.Helper(); t.Fatal("service must not be called") }
	return &fakeAuthService{
		registerFn: func(context.Context, user.User) (user.User, auth.TokenPair, error) {
			fail()
			return user.User{}, auth.TokenPair{}, nil
		},
		loginFn: func(context.Context, string, string) (user.User, auth.TokenPair, error) {
			fail()
			return user.User{}, auth.TokenPair{}, nil
		},
		refreshFn: func(context.Context, string) (auth.TokenPair, error) {
			fail()
			return auth.TokenPair{}, nil
		},
		logoutFn: func(context.Context, string) error {
			fail()
			return nil
		},
	}
}

func TestAuthHandler_Register_ValidPayload_Returns201WithUserAndTokens(t *testing.T) {
	var got user.User
	svc := notCalled(t)
	svc.registerFn = func(_ context.Context, u user.User) (user.User, auth.TokenPair, error) {
		got = u
		return authUser, authTokens, nil
	}

	rec, env := serveAuth(t, svc, "/api/v1/auth/register",
		`{"email":"alice@example.com","password":"s3cret-pass","name":"Alice"}`)

	if rec.Code != nethttp.StatusCreated {
		t.Fatalf("status = %d, want 201 (body %s)", rec.Code, rec.Body.String())
	}
	if got.Email != "alice@example.com" || got.Password != "s3cret-pass" || got.Name != "Alice" || got.ID != "" {
		t.Fatalf("service got %+v", got)
	}
	assertJSONEqual(t, "data", decodeAuthData(t, env), map[string]any{
		"user": map[string]any{
			"id":           "u-1",
			"email":        "alice@example.com",
			"name":         "Alice",
			"role":         "user",
			"total_points": float64(0),
			"created_at":   "2026-10-29T10:00:00Z",
			"updated_at":   "2026-10-29T10:00:00Z",
		},
		"tokens": wantTokensJSON,
	})
	if strings.Contains(rec.Body.String(), "hashed-secret") {
		t.Fatal("password hash leaked in response")
	}
}

func TestAuthHandler_Register_InvalidPayload_Returns400(t *testing.T) {
	for name, body := range map[string]string{
		"empty body":     "",
		"invalid json":   `{"email":`,
		"missing email":  `{"password":"s3cret-pass","name":"Alice"}`,
		"invalid email":  `{"email":"nope","password":"s3cret-pass","name":"Alice"}`,
		"short password": `{"email":"alice@example.com","password":"short","name":"Alice"}`,
		"long password":  `{"email":"alice@example.com","password":"` + strings.Repeat("a", 73) + `","name":"Alice"}`,
		"missing name":   `{"email":"alice@example.com","password":"s3cret-pass"}`,
	} {
		t.Run(name, func(t *testing.T) {
			rec, env := serveAuth(t, notCalled(t), "/api/v1/auth/register", body)
			assertAuthError(t, rec, env, nethttp.StatusBadRequest, "VALIDATION_ERROR")
		})
	}
}

func TestAuthHandler_Register_EmailTaken_Returns409(t *testing.T) {
	svc := notCalled(t)
	svc.registerFn = func(context.Context, user.User) (user.User, auth.TokenPair, error) {
		return user.User{}, auth.TokenPair{}, user.ErrEmailTaken
	}

	rec, env := serveAuth(t, svc, "/api/v1/auth/register",
		`{"email":"alice@example.com","password":"s3cret-pass","name":"Alice"}`)

	assertAuthError(t, rec, env, nethttp.StatusConflict, "CONFLICT")
	if env.Error.Message != "email already taken" {
		t.Fatalf("message = %q", env.Error.Message)
	}
}

func TestAuthHandler_Login_ValidCredentials_Returns200WithUserAndTokens(t *testing.T) {
	var gotEmail, gotPassword string
	svc := notCalled(t)
	svc.loginFn = func(_ context.Context, email, password string) (user.User, auth.TokenPair, error) {
		gotEmail, gotPassword = email, password
		return authUser, authTokens, nil
	}

	rec, env := serveAuth(t, svc, "/api/v1/auth/login", `{"email":"alice@example.com","password":"whatever"}`)

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
	if gotEmail != "alice@example.com" || gotPassword != "whatever" {
		t.Fatalf("service got %q / %q", gotEmail, gotPassword)
	}
	data := decodeAuthData(t, env)
	assertJSONEqual(t, "tokens", data["tokens"], wantTokensJSON)
	if u, ok := data["user"].(map[string]any); !ok || u["id"] != "u-1" {
		t.Fatalf("user = %v", data["user"])
	}
}

func TestAuthHandler_Login_InvalidCredentials_Returns401(t *testing.T) {
	svc := notCalled(t)
	svc.loginFn = func(context.Context, string, string) (user.User, auth.TokenPair, error) {
		return user.User{}, auth.TokenPair{}, apperror.Unauthorized("invalid email or password")
	}

	rec, env := serveAuth(t, svc, "/api/v1/auth/login", `{"email":"alice@example.com","password":"wrong"}`)

	assertAuthError(t, rec, env, nethttp.StatusUnauthorized, "UNAUTHORIZED")
}

func TestAuthHandler_Login_InvalidPayload_Returns400(t *testing.T) {
	rec, env := serveAuth(t, notCalled(t), "/api/v1/auth/login", `{"email":"alice@example.com"}`)
	assertAuthError(t, rec, env, nethttp.StatusBadRequest, "VALIDATION_ERROR")
}

func TestAuthHandler_Refresh_ValidToken_Returns200WithTokens(t *testing.T) {
	var got string
	svc := notCalled(t)
	svc.refreshFn = func(_ context.Context, refreshToken string) (auth.TokenPair, error) {
		got = refreshToken
		return authTokens, nil
	}

	rec, env := serveAuth(t, svc, "/api/v1/auth/refresh", `{"refresh_token":"old-refresh"}`)

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
	if got != "old-refresh" {
		t.Fatalf("service got %q", got)
	}
	assertJSONEqual(t, "data", decodeAuthData(t, env), map[string]any{"tokens": wantTokensJSON})
}

func TestAuthHandler_Refresh_InvalidToken_Returns401(t *testing.T) {
	svc := notCalled(t)
	svc.refreshFn = func(context.Context, string) (auth.TokenPair, error) {
		return auth.TokenPair{}, auth.ErrInvalidRefreshToken
	}

	rec, env := serveAuth(t, svc, "/api/v1/auth/refresh", `{"refresh_token":"revoked"}`)

	assertAuthError(t, rec, env, nethttp.StatusUnauthorized, "UNAUTHORIZED")
	if env.Error.Message != "invalid or expired refresh token" {
		t.Fatalf("message = %q", env.Error.Message)
	}
}

func TestAuthHandler_Refresh_MissingToken_Returns400(t *testing.T) {
	rec, env := serveAuth(t, notCalled(t), "/api/v1/auth/refresh", `{}`)
	assertAuthError(t, rec, env, nethttp.StatusBadRequest, "VALIDATION_ERROR")
	if env.Error.Message != "refresh_token: is required" {
		t.Fatalf("message = %q", env.Error.Message)
	}
}

func TestAuthHandler_Logout_ValidToken_Returns204(t *testing.T) {
	var got string
	svc := notCalled(t)
	svc.logoutFn = func(_ context.Context, refreshToken string) error {
		got = refreshToken
		return nil
	}

	rec, _ := serveAuth(t, svc, "/api/v1/auth/logout", `{"refresh_token":"to-revoke"}`)

	if rec.Code != nethttp.StatusNoContent || rec.Body.Len() != 0 {
		t.Fatalf("status = %d, body = %q, want 204 and empty body", rec.Code, rec.Body.String())
	}
	if got != "to-revoke" {
		t.Fatalf("service got %q", got)
	}
}

func TestAuthHandler_Logout_MissingToken_Returns400(t *testing.T) {
	rec, env := serveAuth(t, notCalled(t), "/api/v1/auth/logout", `{"refresh_token":""}`)
	assertAuthError(t, rec, env, nethttp.StatusBadRequest, "VALIDATION_ERROR")
}
