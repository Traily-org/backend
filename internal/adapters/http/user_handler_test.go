package http

import (
	"context"
	"encoding/json"
	"errors"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/adapters/http/middleware"
	"github.com/traily-org/server/internal/auth"
	"github.com/traily-org/server/internal/domain/user"
)

// testUserHeader carries the ID of the authenticated user in handler tests.
const testUserHeader = "X-Test-User"

type fakeUserService struct {
	getFn    func(ctx context.Context, id string) (user.User, error)
	updateFn func(ctx context.Context, u user.User, currentPassword string) (user.User, error)
	deleteFn func(ctx context.Context, id string) error
}

func (f *fakeUserService) Get(ctx context.Context, id string) (user.User, error) {
	return f.getFn(ctx, id)
}

func (f *fakeUserService) UpdateProfile(ctx context.Context, u user.User, currentPassword string) (user.User, error) {
	return f.updateFn(ctx, u, currentPassword)
}

func (f *fakeUserService) Delete(ctx context.Context, id string) error {
	return f.deleteFn(ctx, id)
}

// noopAuthRoutes satisfies AuthRoutes for tests that only exercise /users.
type noopAuthRoutes struct{}

func (noopAuthRoutes) Register(*gin.Context) {}
func (noopAuthRoutes) Login(*gin.Context)    {}
func (noopAuthRoutes) Refresh(*gin.Context)  {}
func (noopAuthRoutes) Logout(*gin.Context)   {}

// testAuthenticate stores the user named by testUserHeader in the request
// context. Without the header it lets the request through unauthenticated so
// that the handlers' own guard is exercised.
func testAuthenticate(c *gin.Context) {
	if id := c.GetHeader(testUserHeader); id != "" {
		middleware.SetClaims(c, auth.Claims{UserID: id, Role: user.RoleUser})
	}
	c.Next()
}

type envelope struct {
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func init() {
	gin.SetMode(gin.TestMode)
}

var (
	paris       = time.FixedZone("CEST", 2*60*60)
	createdAt   = time.Date(2026, 10, 29, 12, 0, 0, 0, paris)
	sampleUser  = user.User{ID: "u-1", Email: "alice@example.com", Password: "hashed-secret", Name: "Alice", Username: ptr("alice"), DisplayName: ptr("Alice A."), AvatarURL: ptr("https://cdn.example.com/a.png"), Role: user.RoleUser, TotalPoints: 42, CreatedAt: createdAt, UpdatedAt: createdAt}
	errDatabase = errors.New(`ERROR: relation "users" does not exist (SQLSTATE 42P01)`)
)

func ptr(s string) *string { return &s }

// request describes a call made through the server in tests.
type request struct {
	method string
	path   string
	body   string
	userID string
}

func serve(t *testing.T, svc *fakeUserService, r request) (*httptest.ResponseRecorder, envelope) {
	t.Helper()

	srv, err := NewServer(Deps{
		Users:         NewUserHandler(svc),
		Auth:          noopAuthRoutes{},
		Authenticate:  testAuthenticate,
		AuthRateLimit: func(c *gin.Context) { c.Next() },
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	req := httptest.NewRequest(r.method, r.path, strings.NewReader(r.body))
	if r.body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if r.userID != "" {
		req.Header.Set(testUserHeader, r.userID)
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

func assertError(t *testing.T, rec *httptest.ResponseRecorder, env envelope, status int, code, message string) {
	t.Helper()

	if rec.Code != status {
		t.Errorf("status = %d, want %d", rec.Code, status)
	}
	if env.Error == nil {
		t.Fatalf("body %s has no error envelope", rec.Body.String())
	}
	if env.Error.Code != code {
		t.Errorf("error.code = %q, want %q", env.Error.Code, code)
	}
	if message != "" && env.Error.Message != message {
		t.Errorf("error.message = %q, want %q", env.Error.Message, message)
	}
	if env.Data != nil {
		t.Errorf("error response must not contain data, got %s", env.Data)
	}
}

func decodeData(t *testing.T, env envelope) map[string]any {
	t.Helper()

	var got map[string]any
	if err := json.Unmarshal(env.Data, &got); err != nil {
		t.Fatalf("invalid data %s: %v", env.Data, err)
	}
	return got
}

func TestUserHandler_GetMe_AuthenticatedUser_Returns200WithFullProfile(t *testing.T) {
	svc := &fakeUserService{getFn: func(_ context.Context, id string) (user.User, error) {
		if id != "u-1" {
			t.Errorf("id = %q, want u-1", id)
		}
		return sampleUser, nil
	}}

	rec, env := serve(t, svc, request{method: nethttp.MethodGet, path: "/api/v1/users/me", userID: "u-1"})

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
	got := decodeData(t, env)
	want := map[string]any{
		"id":           "u-1",
		"email":        "alice@example.com",
		"name":         "Alice",
		"username":     "alice",
		"display_name": "Alice A.",
		"avatar_url":   "https://cdn.example.com/a.png",
		"role":         "user",
		"total_points": float64(42),
		"created_at":   "2026-10-29T10:00:00Z",
		"updated_at":   "2026-10-29T10:00:00Z",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("data.%s = %v, want %v", k, got[k], v)
		}
	}
	if _, ok := got["password"]; ok {
		t.Error("password must never be exposed")
	}
	if strings.Contains(rec.Body.String(), "hashed-secret") {
		t.Error("body leaks the password")
	}
}

func TestUserHandler_GetMe_NoOptionalFields_OmitsThem(t *testing.T) {
	svc := &fakeUserService{getFn: func(context.Context, string) (user.User, error) {
		return user.User{ID: "u-1", Email: "alice@example.com", Name: "Alice", Role: user.RoleUser}, nil
	}}

	rec, env := serve(t, svc, request{method: nethttp.MethodGet, path: "/api/v1/users/me", userID: "u-1"})

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	got := decodeData(t, env)
	for _, k := range []string{"username", "display_name", "avatar_url"} {
		if _, ok := got[k]; ok {
			t.Errorf("data.%s present, want omitted when unset", k)
		}
	}
	if got["total_points"] != float64(0) {
		t.Errorf("data.total_points = %v, want 0", got["total_points"])
	}
}

func TestUserHandler_MeEndpoints_NoAuthenticatedUser_Returns401(t *testing.T) {
	tests := []struct {
		name   string
		method string
		body   string
	}{
		{"get", nethttp.MethodGet, ""},
		{"update", nethttp.MethodPut, `{"email":"alice@example.com","name":"Alice"}`},
		{"delete", nethttp.MethodDelete, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeUserService{
				getFn: func(context.Context, string) (user.User, error) {
					t.Error("service must not be called without an authenticated user")
					return user.User{}, nil
				},
				updateFn: func(context.Context, user.User, string) (user.User, error) {
					t.Error("service must not be called without an authenticated user")
					return user.User{}, nil
				},
				deleteFn: func(context.Context, string) error {
					t.Error("service must not be called without an authenticated user")
					return nil
				},
			}

			rec, env := serve(t, svc, request{method: tt.method, path: "/api/v1/users/me", body: tt.body})

			assertError(t, rec, env, nethttp.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		})
	}
}

func TestUserHandler_GetMe_UnexpectedError_Returns500WithoutDetails(t *testing.T) {
	svc := &fakeUserService{getFn: func(context.Context, string) (user.User, error) {
		return user.User{}, errDatabase
	}}

	rec, env := serve(t, svc, request{method: nethttp.MethodGet, path: "/api/v1/users/me", userID: "u-1"})

	assertError(t, rec, env, nethttp.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	if strings.Contains(rec.Body.String(), "relation") {
		t.Errorf("body leaks internal error: %s", rec.Body.String())
	}
}

func TestUserHandler_UpdateMe_ValidPayload_Returns200AndPassesFields(t *testing.T) {
	svc := &fakeUserService{updateFn: func(_ context.Context, u user.User, _ string) (user.User, error) {
		if u.ID != "u-1" || u.Email != "alice@example.com" || u.Password != "" || u.Name != "Alice B" {
			t.Errorf("service received %+v", u)
		}
		if u.Username == nil || *u.Username != "alice.b_2-x" {
			t.Errorf("Username = %v, want alice.b_2-x", u.Username)
		}
		if u.DisplayName == nil || *u.DisplayName != "Alice B." {
			t.Errorf("DisplayName = %v, want Alice B.", u.DisplayName)
		}
		if u.AvatarURL == nil || *u.AvatarURL != "https://cdn.example.com/b.png" {
			t.Errorf("AvatarURL = %v, want https://cdn.example.com/b.png", u.AvatarURL)
		}
		if u.Role != "" || u.TotalPoints != 0 {
			t.Errorf("role and total points must not be settable, got %q / %d", u.Role, u.TotalPoints)
		}
		return sampleUser, nil
	}}

	rec, env := serve(t, svc, request{
		method: nethttp.MethodPut,
		path:   "/api/v1/users/me",
		userID: "u-1",
		body:   `{"email":"alice@example.com","name":"Alice B","username":"alice.b_2-x","display_name":"Alice B.","avatar_url":"https://cdn.example.com/b.png","role":"admin","total_points":9999}`,
	})

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
	got := decodeData(t, env)
	if got["id"] != "u-1" || got["email"] != "alice@example.com" || got["role"] != "user" {
		t.Errorf("data = %v, want full sample user", got)
	}
}

func TestUserHandler_UpdateMe_OmittedOptionalFields_PassesNil(t *testing.T) {
	svc := &fakeUserService{updateFn: func(_ context.Context, u user.User, _ string) (user.User, error) {
		if u.Username != nil || u.DisplayName != nil || u.AvatarURL != nil {
			t.Errorf("service received %+v, want nil optional fields", u)
		}
		return sampleUser, nil
	}}

	rec, _ := serve(t, svc, request{
		method: nethttp.MethodPut,
		path:   "/api/v1/users/me",
		userID: "u-1",
		body:   `{"email":"alice@example.com","name":"Alice"}`,
	})

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
}

func TestUserHandler_UpdateMe_InvalidPayload_Returns400(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		message string
	}{
		{"invalid email", `{"email":"nope","name":"Alice"}`, "email: must be a valid email"},
		{"missing name", `{"email":"alice@example.com"}`, "name: is required"},
		{"short password", `{"email":"alice@example.com","password":"short","name":"Alice"}`, "password: must be at least 8 characters"},
		{"long password", `{"email":"alice@example.com","password":"` + strings.Repeat("a", 73) + `","name":"Alice"}`, "password: must be at most 72 characters"},
		{"short username", `{"email":"alice@example.com","name":"Alice","username":"al"}`, "username: must be at least 3 characters"},
		{"long username", `{"email":"alice@example.com","name":"Alice","username":"` + strings.Repeat("a", 51) + `"}`, "username: must be at most 50 characters"},
		{"username with space", `{"email":"alice@example.com","name":"Alice","username":"alice b"}`, "username: is invalid"},
		{"username with symbol", `{"email":"alice@example.com","name":"Alice","username":"alice@home"}`, "username: is invalid"},
		{"long display name", `{"email":"alice@example.com","name":"Alice","display_name":"` + strings.Repeat("a", 101) + `"}`, "display_name: must be at most 100 characters"},
		{"invalid avatar url", `{"email":"alice@example.com","name":"Alice","avatar_url":"not a url"}`, "avatar_url: must be a valid URL"},
		{"javascript avatar url", `{"email":"alice@example.com","name":"Alice","avatar_url":"javascript:alert(1)"}`, "avatar_url: must be a valid URL"},
		{"data avatar url", `{"email":"alice@example.com","name":"Alice","avatar_url":"data:text/html,hi"}`, "avatar_url: must be a valid URL"},
		{"malformed JSON", `{"email":`, "request body is not valid JSON"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeUserService{updateFn: func(context.Context, user.User, string) (user.User, error) {
				t.Error("service must not be called on invalid payload")
				return user.User{}, nil
			}}

			rec, env := serve(t, svc, request{method: nethttp.MethodPut, path: "/api/v1/users/me", userID: "u-1", body: tt.body})

			assertError(t, rec, env, nethttp.StatusBadRequest, "VALIDATION_ERROR", tt.message)
		})
	}
}

func TestUserHandler_UpdateMe_PasswordTooLong_Returns400(t *testing.T) {
	svc := &fakeUserService{updateFn: func(context.Context, user.User, string) (user.User, error) {
		return user.User{}, user.ErrPasswordTooLong
	}}

	// 40 characters, 80 bytes: passes the binding, rejected by the domain.
	rec, env := serve(t, svc, request{
		method: nethttp.MethodPut,
		path:   "/api/v1/users/me",
		userID: "u-1",
		body:   `{"email":"alice@example.com","password":"` + strings.Repeat("é", 40) + `","name":"Alice"}`,
	})

	assertError(t, rec, env, nethttp.StatusBadRequest, "VALIDATION_ERROR", "password must be at most 72 bytes")
}

func TestUserHandler_UpdateMe_Conflict_Returns409(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		message string
	}{
		{"email taken", user.ErrEmailTaken, "email already taken"},
		{"username taken", user.ErrUsernameTaken, "username already taken"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeUserService{updateFn: func(context.Context, user.User, string) (user.User, error) {
				return user.User{}, tt.err
			}}

			rec, env := serve(t, svc, request{
				method: nethttp.MethodPut,
				path:   "/api/v1/users/me",
				userID: "u-1",
				body:   `{"email":"bob@example.com","name":"Alice","username":"bob"}`,
			})

			assertError(t, rec, env, nethttp.StatusConflict, "CONFLICT", tt.message)
		})
	}
}

func TestUserHandler_UpdateMe_UnknownUser_Returns404(t *testing.T) {
	svc := &fakeUserService{updateFn: func(context.Context, user.User, string) (user.User, error) {
		return user.User{}, user.ErrNotFound
	}}

	rec, env := serve(t, svc, request{
		method: nethttp.MethodPut,
		path:   "/api/v1/users/me",
		userID: "u-1",
		body:   `{"email":"alice@example.com","name":"Alice"}`,
	})

	assertError(t, rec, env, nethttp.StatusNotFound, "NOT_FOUND", "user not found")
}

func TestUserHandler_DeleteMe_AuthenticatedUser_Returns204(t *testing.T) {
	svc := &fakeUserService{deleteFn: func(_ context.Context, id string) error {
		if id != "u-1" {
			t.Errorf("id = %q, want u-1", id)
		}
		return nil
	}}

	rec, _ := serve(t, svc, request{method: nethttp.MethodDelete, path: "/api/v1/users/me", userID: "u-1"})

	if rec.Code != nethttp.StatusNoContent {
		t.Errorf("status = %d, want 204", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", rec.Body.String())
	}
}

func TestUserHandler_DeleteMe_UnexpectedError_Returns500WithoutDetails(t *testing.T) {
	svc := &fakeUserService{deleteFn: func(context.Context, string) error {
		return errDatabase
	}}

	rec, env := serve(t, svc, request{method: nethttp.MethodDelete, path: "/api/v1/users/me", userID: "u-1"})

	assertError(t, rec, env, nethttp.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	if strings.Contains(rec.Body.String(), "SQLSTATE") {
		t.Errorf("body leaks internal error: %s", rec.Body.String())
	}
}

func TestUserHandler_Get_ExistingUser_Returns200WithPublicProfile(t *testing.T) {
	svc := &fakeUserService{getFn: func(_ context.Context, id string) (user.User, error) {
		if id != "u-2" {
			t.Errorf("id = %q, want u-2", id)
		}
		return sampleUser, nil
	}}

	rec, env := serve(t, svc, request{method: nethttp.MethodGet, path: "/api/v1/users/u-2", userID: "u-1"})

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
	got := decodeData(t, env)
	want := map[string]any{
		"id":           "u-1",
		"username":     "alice",
		"display_name": "Alice A.",
		"avatar_url":   "https://cdn.example.com/a.png",
		"total_points": float64(42),
		"created_at":   "2026-10-29T10:00:00Z",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("data.%s = %v, want %v", k, got[k], v)
		}
	}
	for _, k := range []string{"email", "role", "name", "password", "updated_at"} {
		if _, ok := got[k]; ok {
			t.Errorf("public profile exposes %q", k)
		}
	}
	if strings.Contains(rec.Body.String(), "alice@example.com") {
		t.Error("public profile leaks the email")
	}
}

func TestUserHandler_Get_UnknownUser_Returns404(t *testing.T) {
	svc := &fakeUserService{getFn: func(context.Context, string) (user.User, error) {
		return user.User{}, user.ErrNotFound
	}}

	rec, env := serve(t, svc, request{method: nethttp.MethodGet, path: "/api/v1/users/missing", userID: "u-1"})

	assertError(t, rec, env, nethttp.StatusNotFound, "NOT_FOUND", "user not found")
}

func TestServer_UnknownRoute_Returns404Envelope(t *testing.T) {
	rec, env := serve(t, &fakeUserService{}, request{method: nethttp.MethodGet, path: "/api/v1/nope"})

	assertError(t, rec, env, nethttp.StatusNotFound, "NOT_FOUND", "route not found")
}

func TestServer_WrongMethod_Returns405Envelope(t *testing.T) {
	rec, env := serve(t, &fakeUserService{}, request{method: nethttp.MethodPatch, path: "/api/v1/users/me", userID: "u-1"})

	assertError(t, rec, env, nethttp.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
}

func TestServer_HandlerPanics_Returns500Envelope(t *testing.T) {
	svc := &fakeUserService{getFn: func(context.Context, string) (user.User, error) {
		panic("boom: secret detail")
	}}

	rec, env := serve(t, svc, request{method: nethttp.MethodGet, path: "/api/v1/users/u-1", userID: "u-1"})

	assertError(t, rec, env, nethttp.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	if strings.Contains(rec.Body.String(), "secret") {
		t.Errorf("body leaks panic value: %s", rec.Body.String())
	}
}
