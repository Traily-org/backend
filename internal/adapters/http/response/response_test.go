package response_test

import (
	"encoding/json"
	"errors"
	"fmt"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/adapters/http/response"
	"github.com/traily-org/server/internal/pkg/apperror"
)

type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func init() {
	gin.SetMode(gin.TestMode)
}

func newContext(body string) (*gin.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(nethttp.MethodPost, "/test", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	return c, rec
}

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) errorEnvelope {
	t.Helper()
	var env errorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON body %q: %v", rec.Body.String(), err)
	}
	return env
}

func TestJSON_WithData_WrapsInDataEnvelope(t *testing.T) {
	c, rec := newContext("")

	response.JSON(c, nethttp.StatusCreated, map[string]string{"id": "42"})

	if rec.Code != nethttp.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, nethttp.StatusCreated)
	}
	if got, want := rec.Body.String(), `{"data":{"id":"42"}}`; got != want {
		t.Errorf("body = %s, want %s", got, want)
	}
}

func TestNoContent_Called_Returns204WithEmptyBody(t *testing.T) {
	c, rec := newContext("")

	response.NoContent(c)
	c.Writer.WriteHeaderNow()

	if rec.Code != nethttp.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, nethttp.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", rec.Body.String())
	}
}

func TestError_AppErrors_UsesCodeStatusAndMessage(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		code   string
		msg    string
	}{
		{"validation", apperror.Validation("email is required"), nethttp.StatusBadRequest, "VALIDATION_ERROR", "email is required"},
		{"unauthorized", apperror.Unauthorized("missing token"), nethttp.StatusUnauthorized, "UNAUTHORIZED", "missing token"},
		{"forbidden", apperror.Forbidden("not allowed"), nethttp.StatusForbidden, "FORBIDDEN", "not allowed"},
		{"not found", apperror.NotFound("user not found"), nethttp.StatusNotFound, "NOT_FOUND", "user not found"},
		{"conflict", apperror.Conflict("email already taken"), nethttp.StatusConflict, "CONFLICT", "email already taken"},
		{"rate limited", apperror.RateLimited("slow down"), nethttp.StatusTooManyRequests, "RATE_LIMITED", "slow down"},
		{"wrapped", fmt.Errorf("service: %w", apperror.NotFound("user not found")), nethttp.StatusNotFound, "NOT_FOUND", "user not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, rec := newContext("")

			response.Error(c, tt.err)

			if rec.Code != tt.status {
				t.Errorf("status = %d, want %d", rec.Code, tt.status)
			}
			env := decodeError(t, rec)
			if env.Error.Code != tt.code || env.Error.Message != tt.msg {
				t.Errorf("error = %+v, want {%s %s}", env.Error, tt.code, tt.msg)
			}
			if !c.IsAborted() {
				t.Error("context should be aborted")
			}
		})
	}
}

func TestError_InternalOrUnknownError_MasksDetails(t *testing.T) {
	secret := "pq: relation \"users\" does not exist"
	tests := []struct {
		name string
		err  error
	}{
		{"plain error", errors.New(secret)},
		{"internal app error", apperror.Internal(errors.New(secret))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, rec := newContext("")

			response.Error(c, tt.err)

			if rec.Code != nethttp.StatusInternalServerError {
				t.Errorf("status = %d, want 500", rec.Code)
			}
			env := decodeError(t, rec)
			if env.Error.Code != "INTERNAL_ERROR" || env.Error.Message != "internal server error" {
				t.Errorf("error = %+v, want generic internal error", env.Error)
			}
			if strings.Contains(rec.Body.String(), "relation") {
				t.Errorf("body leaks internal error: %s", rec.Body.String())
			}
		})
	}
}

func TestWriteError_ExplicitStatus_WritesEnvelope(t *testing.T) {
	c, rec := newContext("")

	response.WriteError(c, nethttp.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")

	if rec.Code != nethttp.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
	if got, want := rec.Body.String(), `{"error":{"code":"METHOD_NOT_ALLOWED","message":"method not allowed"}}`; got != want {
		t.Errorf("body = %s, want %s", got, want)
	}
}

type bindPayload struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"omitempty,min=8"`
	Age      int    `json:"age" binding:"omitempty,gte=18"`
	Role     string `json:"role" binding:"omitempty,oneof=admin member"`
}

func TestBindError_InvalidPayloads_ReturnsReadableValidationError(t *testing.T) {
	tests := []struct {
		name string
		body string
		msg  string
	}{
		{"missing field", `{}`, "email: is required"},
		{"invalid email", `{"email":"nope"}`, "email: must be a valid email"},
		{"too short", `{"email":"a@b.co","password":"short"}`, "password: must be at least 8 characters"},
		{"numeric bound", `{"email":"a@b.co","age":12}`, "age: must be greater than or equal to 18"},
		{"oneof", `{"email":"a@b.co","role":"root"}`, "role: must be one of: admin, member"},
		{"several errors", `{"email":"nope","password":"short"}`, "email: must be a valid email; password: must be at least 8 characters"},
		{"empty body", ``, "request body is required"},
		{"malformed JSON", `{"email":`, "request body is not valid JSON"},
		{"syntax error", `{"email" "x"}`, "request body is not valid JSON"},
		{"wrong type", `{"email":42}`, "email: must be of type string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, rec := newContext(tt.body)

			var payload bindPayload
			err := c.ShouldBindJSON(&payload)
			if err == nil {
				t.Fatal("ShouldBindJSON() error = nil, want error")
			}
			response.BindError(c, err)

			if rec.Code != nethttp.StatusBadRequest {
				t.Errorf("status = %d, want 400", rec.Code)
			}
			env := decodeError(t, rec)
			if env.Error.Code != "VALIDATION_ERROR" {
				t.Errorf("code = %q, want VALIDATION_ERROR", env.Error.Code)
			}
			if env.Error.Message != tt.msg {
				t.Errorf("message = %q, want %q", env.Error.Message, tt.msg)
			}
		})
	}
}

func TestBindError_UnknownError_ReturnsGenericMessage(t *testing.T) {
	c, rec := newContext("")

	response.BindError(c, errors.New("something odd"))

	env := decodeError(t, rec)
	if rec.Code != nethttp.StatusBadRequest || env.Error.Message != "invalid request body" {
		t.Errorf("got %d %+v, want 400 invalid request body", rec.Code, env.Error)
	}
}
