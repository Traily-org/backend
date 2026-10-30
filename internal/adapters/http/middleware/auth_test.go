package middleware

import (
	"encoding/json"
	"errors"
	nethttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/auth"
	"github.com/traily-org/server/internal/domain/user"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// fakeParser accepts validToken and rejects anything else with err.
type fakeParser struct {
	validToken string
	claims     auth.Claims
	err        error
	gotToken   string
}

func (f *fakeParser) ParseAccessToken(token string) (auth.Claims, error) {
	f.gotToken = token
	if token == f.validToken {
		return f.claims, nil
	}
	return auth.Claims{}, f.err
}

type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// seenClaims records what the protected handler observed.
type seenClaims struct {
	called bool
	userID string
	role   user.Role
}

func newAuthRouter(seen *seenClaims, middlewares ...gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	handlers := append(middlewares, func(c *gin.Context) {
		seen.called = true
		seen.userID, _ = UserID(c)
		seen.role, _ = UserRole(c)
		c.Status(nethttp.StatusOK)
	})
	r.GET("/protected", handlers...)
	return r
}

func doGet(t *testing.T, r *gin.Engine, authorization string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(nethttp.MethodGet, "/protected", nil)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func assertErrorResponse(t *testing.T, rec *httptest.ResponseRecorder, status int, code, message string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d (body %s)", rec.Code, status, rec.Body.String())
	}
	var env errorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON body %q: %v", rec.Body.String(), err)
	}
	if env.Error.Code != code || env.Error.Message != message {
		t.Fatalf("error = %+v, want %s %q", env.Error, code, message)
	}
}

func newFakeParser() *fakeParser {
	return &fakeParser{
		validToken: "good-token",
		claims:     auth.Claims{UserID: "u-1", Role: user.RoleModerator},
		err:        auth.ErrInvalidToken,
	}
}

func TestAuthenticate_NoHeader_Returns401(t *testing.T) {
	seen := &seenClaims{}
	rec := doGet(t, newAuthRouter(seen, Authenticate(newFakeParser())), "")

	assertErrorResponse(t, rec, nethttp.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	if seen.called {
		t.Fatal("handler called without authentication")
	}
}

func TestAuthenticate_MalformedHeader_Returns401(t *testing.T) {
	for _, header := range []string{
		"Basic Zm9vOmJhcg==",
		"Bearer",
		"Bearer ",
		"Bearergood-token",
		"Bearer  good-token",
		"Bearer good token",
		"Token good-token",
	} {
		t.Run(header, func(t *testing.T) {
			seen := &seenClaims{}
			rec := doGet(t, newAuthRouter(seen, Authenticate(newFakeParser())), header)

			assertErrorResponse(t, rec, nethttp.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
			if seen.called {
				t.Fatal("handler called with a malformed header")
			}
		})
	}
}

func TestAuthenticate_RejectedToken_Returns401WithParserError(t *testing.T) {
	seen := &seenClaims{}
	rec := doGet(t, newAuthRouter(seen, Authenticate(newFakeParser())), "Bearer forged-token")

	assertErrorResponse(t, rec, nethttp.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired access token")
	if seen.called {
		t.Fatal("handler called with a rejected token")
	}
}

func TestAuthenticate_ParserFailsUnexpectedly_Returns500WithoutDetails(t *testing.T) {
	p := newFakeParser()
	p.err = errors.New("key store unreachable")
	rec := doGet(t, newAuthRouter(&seenClaims{}, Authenticate(p)), "Bearer other-token")

	assertErrorResponse(t, rec, nethttp.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}

func TestAuthenticate_ValidToken_CallsHandlerWithClaims(t *testing.T) {
	for _, header := range []string{"Bearer good-token", "bearer good-token", "BEARER good-token", " Bearer good-token "} {
		t.Run(header, func(t *testing.T) {
			p := newFakeParser()
			seen := &seenClaims{}
			rec := doGet(t, newAuthRouter(seen, Authenticate(p)), header)

			if rec.Code != nethttp.StatusOK {
				t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
			}
			if p.gotToken != "good-token" {
				t.Fatalf("parsed token = %q, want good-token", p.gotToken)
			}
			if !seen.called || seen.userID != "u-1" || seen.role != user.RoleModerator {
				t.Fatalf("handler saw %+v, want u-1 moderator", seen)
			}
		})
	}
}

func TestRequireRole_NotAuthenticated_Returns401(t *testing.T) {
	seen := &seenClaims{}
	rec := doGet(t, newAuthRouter(seen, RequireRole(user.RoleAdmin)), "")

	assertErrorResponse(t, rec, nethttp.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	if seen.called {
		t.Fatal("handler called without authentication")
	}
}

func TestRequireRole_RoleNotAllowed_Returns403(t *testing.T) {
	seen := &seenClaims{}
	r := newAuthRouter(seen, Authenticate(newFakeParser()), RequireRole(user.RoleAdmin))
	rec := doGet(t, r, "Bearer good-token")

	assertErrorResponse(t, rec, nethttp.StatusForbidden, "FORBIDDEN", "insufficient permissions")
	if seen.called {
		t.Fatal("handler called with an insufficient role")
	}
}

func TestRequireRole_RoleAllowed_CallsHandler(t *testing.T) {
	seen := &seenClaims{}
	r := newAuthRouter(seen, Authenticate(newFakeParser()), RequireRole(user.RoleModerator, user.RoleAdmin))
	rec := doGet(t, r, "Bearer good-token")

	if rec.Code != nethttp.StatusOK || !seen.called {
		t.Fatalf("status = %d, called = %v, want 200 and handler called", rec.Code, seen.called)
	}
}
