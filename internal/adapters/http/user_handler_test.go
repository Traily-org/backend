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

	"github.com/traily-org/server/internal/domain/user"
)

type fakeUserService struct {
	getFn    func(ctx context.Context, id string) (user.User, error)
	createFn func(ctx context.Context, u user.User) (user.User, error)
	updateFn func(ctx context.Context, u user.User) (user.User, error)
	deleteFn func(ctx context.Context, id string) error
}

func (f *fakeUserService) Get(ctx context.Context, id string) (user.User, error) {
	return f.getFn(ctx, id)
}

func (f *fakeUserService) Create(ctx context.Context, u user.User) (user.User, error) {
	return f.createFn(ctx, u)
}

func (f *fakeUserService) Update(ctx context.Context, u user.User) (user.User, error) {
	return f.updateFn(ctx, u)
}

func (f *fakeUserService) Delete(ctx context.Context, id string) error {
	return f.deleteFn(ctx, id)
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
	sampleUser  = user.User{ID: "u-1", Email: "alice@example.com", Password: "hashed-secret", Name: "Alice", CreatedAt: createdAt, UpdatedAt: createdAt}
	errDatabase = errors.New(`ERROR: relation "users" does not exist (SQLSTATE 42P01)`)
)

func serve(t *testing.T, svc *fakeUserService, method, path, body string) (*httptest.ResponseRecorder, envelope) {
	t.Helper()

	server := NewServer(NewUserHandler(svc))
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

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

func decodeUser(t *testing.T, env envelope) map[string]any {
	t.Helper()

	var got map[string]any
	if err := json.Unmarshal(env.Data, &got); err != nil {
		t.Fatalf("invalid data %s: %v", env.Data, err)
	}
	return got
}

func TestUserHandler_Get_ExistingUser_Returns200WithDataEnvelope(t *testing.T) {
	svc := &fakeUserService{getFn: func(_ context.Context, id string) (user.User, error) {
		if id != "u-1" {
			t.Errorf("id = %q, want u-1", id)
		}
		return sampleUser, nil
	}}

	rec, env := serve(t, svc, nethttp.MethodGet, "/api/v1/users/u-1", "")

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if env.Error != nil {
		t.Fatalf("unexpected error envelope: %+v", env.Error)
	}
	got := decodeUser(t, env)
	if got["id"] != "u-1" || got["email"] != "alice@example.com" || got["name"] != "Alice" {
		t.Errorf("data = %v, want sample user", got)
	}
	if got["created_at"] != "2026-10-29T10:00:00Z" {
		t.Errorf("created_at = %v, want UTC ISO 8601 2026-10-29T10:00:00Z", got["created_at"])
	}
	if _, ok := got["password"]; ok {
		t.Error("password must never be exposed")
	}
	if strings.Contains(rec.Body.String(), "hashed-secret") {
		t.Error("body leaks the password")
	}
}

func TestUserHandler_Get_UnknownUser_Returns404(t *testing.T) {
	svc := &fakeUserService{getFn: func(context.Context, string) (user.User, error) {
		return user.User{}, user.ErrNotFound
	}}

	rec, env := serve(t, svc, nethttp.MethodGet, "/api/v1/users/missing", "")

	assertError(t, rec, env, nethttp.StatusNotFound, "NOT_FOUND", "user not found")
}

func TestUserHandler_Get_UnexpectedError_Returns500WithoutDetails(t *testing.T) {
	svc := &fakeUserService{getFn: func(context.Context, string) (user.User, error) {
		return user.User{}, errDatabase
	}}

	rec, env := serve(t, svc, nethttp.MethodGet, "/api/v1/users/u-1", "")

	assertError(t, rec, env, nethttp.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	if strings.Contains(rec.Body.String(), "relation") {
		t.Errorf("body leaks internal error: %s", rec.Body.String())
	}
}

func TestUserHandler_Create_ValidPayload_Returns201(t *testing.T) {
	svc := &fakeUserService{createFn: func(_ context.Context, u user.User) (user.User, error) {
		if u.Email != "alice@example.com" || u.Password != "supersecret" || u.Name != "Alice" {
			t.Errorf("service received %+v", u)
		}
		return sampleUser, nil
	}}

	rec, env := serve(t, svc, nethttp.MethodPost, "/api/v1/users",
		`{"email":"alice@example.com","password":"supersecret","name":"Alice"}`)

	if rec.Code != nethttp.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}
	if got := decodeUser(t, env); got["id"] != "u-1" {
		t.Errorf("data.id = %v, want u-1", got["id"])
	}
}

func TestUserHandler_Create_InvalidPayload_Returns400(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		message string
	}{
		{"invalid email", `{"email":"nope","password":"supersecret","name":"Alice"}`, "email: must be a valid email"},
		{"missing name", `{"email":"alice@example.com","password":"supersecret"}`, "name: is required"},
		{"short password", `{"email":"alice@example.com","password":"short","name":"Alice"}`, "password: must be at least 8 characters"},
		{"long password", `{"email":"alice@example.com","password":"` + strings.Repeat("a", 73) + `","name":"Alice"}`, "password: must be at most 72 characters"},
		{"malformed JSON", `{"email":`, "request body is not valid JSON"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeUserService{createFn: func(context.Context, user.User) (user.User, error) {
				t.Error("service must not be called on invalid payload")
				return user.User{}, nil
			}}

			rec, env := serve(t, svc, nethttp.MethodPost, "/api/v1/users", tt.body)

			assertError(t, rec, env, nethttp.StatusBadRequest, "VALIDATION_ERROR", tt.message)
		})
	}
}

func TestUserHandler_Create_PasswordTooLong_Returns400(t *testing.T) {
	svc := &fakeUserService{createFn: func(context.Context, user.User) (user.User, error) {
		return user.User{}, user.ErrPasswordTooLong
	}}

	// 40 characters, 80 bytes: passes the binding, rejected by the domain.
	rec, env := serve(t, svc, nethttp.MethodPost, "/api/v1/users",
		`{"email":"alice@example.com","password":"`+strings.Repeat("é", 40)+`","name":"Alice"}`)

	assertError(t, rec, env, nethttp.StatusBadRequest, "VALIDATION_ERROR", "password must be at most 72 bytes")
}

func TestUserHandler_Create_EmailTaken_Returns409(t *testing.T) {
	svc := &fakeUserService{createFn: func(context.Context, user.User) (user.User, error) {
		return user.User{}, user.ErrEmailTaken
	}}

	rec, env := serve(t, svc, nethttp.MethodPost, "/api/v1/users",
		`{"email":"alice@example.com","password":"supersecret","name":"Alice"}`)

	assertError(t, rec, env, nethttp.StatusConflict, "CONFLICT", "email already taken")
}

func TestUserHandler_Update_WithoutPassword_Returns200AndPassesEmptyPassword(t *testing.T) {
	svc := &fakeUserService{updateFn: func(_ context.Context, u user.User) (user.User, error) {
		if u.ID != "u-1" || u.Password != "" || u.Name != "Alice B" {
			t.Errorf("service received %+v", u)
		}
		return sampleUser, nil
	}}

	rec, env := serve(t, svc, nethttp.MethodPut, "/api/v1/users/u-1",
		`{"email":"alice@example.com","name":"Alice B"}`)

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
	if got := decodeUser(t, env); got["id"] != "u-1" {
		t.Errorf("data.id = %v, want u-1", got["id"])
	}
}

func TestUserHandler_Update_ShortPassword_Returns400(t *testing.T) {
	svc := &fakeUserService{updateFn: func(context.Context, user.User) (user.User, error) {
		t.Error("service must not be called on invalid payload")
		return user.User{}, nil
	}}

	rec, env := serve(t, svc, nethttp.MethodPut, "/api/v1/users/u-1",
		`{"email":"alice@example.com","password":"short","name":"Alice"}`)

	assertError(t, rec, env, nethttp.StatusBadRequest, "VALIDATION_ERROR", "password: must be at least 8 characters")
}

func TestUserHandler_Update_LongPassword_Returns400(t *testing.T) {
	svc := &fakeUserService{updateFn: func(context.Context, user.User) (user.User, error) {
		t.Error("service must not be called on invalid payload")
		return user.User{}, nil
	}}

	rec, env := serve(t, svc, nethttp.MethodPut, "/api/v1/users/u-1",
		`{"email":"alice@example.com","password":"`+strings.Repeat("a", 73)+`","name":"Alice"}`)

	assertError(t, rec, env, nethttp.StatusBadRequest, "VALIDATION_ERROR", "password: must be at most 72 characters")
}

func TestUserHandler_Update_UnknownUser_Returns404(t *testing.T) {
	svc := &fakeUserService{updateFn: func(context.Context, user.User) (user.User, error) {
		return user.User{}, user.ErrNotFound
	}}

	rec, env := serve(t, svc, nethttp.MethodPut, "/api/v1/users/missing",
		`{"email":"alice@example.com","name":"Alice"}`)

	assertError(t, rec, env, nethttp.StatusNotFound, "NOT_FOUND", "user not found")
}

func TestUserHandler_Update_EmailTaken_Returns409(t *testing.T) {
	svc := &fakeUserService{updateFn: func(context.Context, user.User) (user.User, error) {
		return user.User{}, user.ErrEmailTaken
	}}

	rec, env := serve(t, svc, nethttp.MethodPut, "/api/v1/users/u-1",
		`{"email":"bob@example.com","name":"Alice"}`)

	assertError(t, rec, env, nethttp.StatusConflict, "CONFLICT", "email already taken")
}

func TestUserHandler_Delete_ExistingUser_Returns204(t *testing.T) {
	svc := &fakeUserService{deleteFn: func(_ context.Context, id string) error {
		if id != "u-1" {
			t.Errorf("id = %q, want u-1", id)
		}
		return nil
	}}

	rec, _ := serve(t, svc, nethttp.MethodDelete, "/api/v1/users/u-1", "")

	if rec.Code != nethttp.StatusNoContent {
		t.Errorf("status = %d, want 204", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", rec.Body.String())
	}
}

func TestUserHandler_Delete_UnknownUser_Returns404(t *testing.T) {
	svc := &fakeUserService{deleteFn: func(context.Context, string) error {
		return user.ErrNotFound
	}}

	rec, env := serve(t, svc, nethttp.MethodDelete, "/api/v1/users/missing", "")

	assertError(t, rec, env, nethttp.StatusNotFound, "NOT_FOUND", "user not found")
}

func TestUserHandler_Delete_UnexpectedError_Returns500WithoutDetails(t *testing.T) {
	svc := &fakeUserService{deleteFn: func(context.Context, string) error {
		return errDatabase
	}}

	rec, env := serve(t, svc, nethttp.MethodDelete, "/api/v1/users/u-1", "")

	assertError(t, rec, env, nethttp.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	if strings.Contains(rec.Body.String(), "SQLSTATE") {
		t.Errorf("body leaks internal error: %s", rec.Body.String())
	}
}

func TestServer_UnknownRoute_Returns404Envelope(t *testing.T) {
	rec, env := serve(t, &fakeUserService{}, nethttp.MethodGet, "/api/v1/nope", "")

	assertError(t, rec, env, nethttp.StatusNotFound, "NOT_FOUND", "route not found")
}

func TestServer_WrongMethod_Returns405Envelope(t *testing.T) {
	rec, env := serve(t, &fakeUserService{}, nethttp.MethodPatch, "/api/v1/users/u-1", "")

	assertError(t, rec, env, nethttp.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
}

func TestServer_HandlerPanics_Returns500Envelope(t *testing.T) {
	svc := &fakeUserService{getFn: func(context.Context, string) (user.User, error) {
		panic("boom: secret detail")
	}}

	rec, env := serve(t, svc, nethttp.MethodGet, "/api/v1/users/u-1", "")

	assertError(t, rec, env, nethttp.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	if strings.Contains(rec.Body.String(), "secret") {
		t.Errorf("body leaks panic value: %s", rec.Body.String())
	}
}
