//go:build integration

package http

import (
	"encoding/json"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/traily-org/server/internal/domain/trail"
	"github.com/traily-org/server/internal/domain/user"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/internal/testutil"
)

// newTrailAPI builds the real stack (HTTP, services, PostgreSQL) and returns a
// helper sending requests to it, plus the id of an existing user.
func newTrailAPI(t *testing.T) (func(method, path, body string) (*httptest.ResponseRecorder, envelope), string) {
	t.Helper()

	pool := testutil.NewPool(t)
	server := NewServer(
		NewUserHandler(user.NewService(postgres.NewUserRepository(pool))),
		NewTrailHandler(trail.NewService(postgres.NewTrailRepository(pool))),
	)

	send := func(method, path, body string) (*httptest.ResponseRecorder, envelope) {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, req)

		var env envelope
		if rec.Body.Len() > 0 {
			if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
				t.Fatalf("response body is not valid JSON: %v (%s)", err, rec.Body.String())
			}
		}
		return rec, env
	}

	return send, testutil.CreateUser(t, pool)
}

func TestTrailAPI_SaveThenConsult(t *testing.T) {
	send, userID := newTrailAPI(t)

	body := strings.Replace(createTrailBody, sampleUserID, userID, 1)
	rec, env := send(nethttp.MethodPost, "/api/v1/trails", body)
	if rec.Code != nethttp.StatusCreated {
		t.Fatalf("POST /trails status = %d, want %d (%s)", rec.Code, nethttp.StatusCreated, rec.Body.String())
	}

	var created trailResponse
	if err := json.Unmarshal(env.Data, &created); err != nil {
		t.Fatalf("POST /trails data: %v", err)
	}
	if created.ID == "" || created.UserID == nil || *created.UserID != userID {
		t.Fatalf("created trail = %+v", created)
	}

	rec, env = send(nethttp.MethodGet, "/api/v1/trails/"+created.ID, "")
	if rec.Code != nethttp.StatusOK {
		t.Fatalf("GET /trails/:id status = %d, want %d", rec.Code, nethttp.StatusOK)
	}
	var fetched trailResponse
	if err := json.Unmarshal(env.Data, &fetched); err != nil {
		t.Fatalf("GET /trails/:id data: %v", err)
	}
	if fetched.Name != "Tour du lac" || fetched.Difficulty != "easy" || fetched.DistanceMeters != 4200.5 {
		t.Fatalf("fetched trail = %+v", fetched)
	}
	if len(fetched.Path.Coordinates) != 2 || fetched.Path.Coordinates[0][0] != 6.1294 || fetched.Path.Coordinates[1][1] != 45.9004 {
		t.Fatalf("fetched path = %+v", fetched.Path)
	}

	rec, env = send(nethttp.MethodGet, "/api/v1/trails?user_id="+userID, "")
	if rec.Code != nethttp.StatusOK {
		t.Fatalf("GET /trails status = %d, want %d", rec.Code, nethttp.StatusOK)
	}
	var listed []trailResponse
	if err := json.Unmarshal(env.Data, &listed); err != nil || len(listed) != 1 || listed[0].ID != created.ID {
		t.Fatalf("GET /trails data = %s (err %v), want the saved trail", env.Data, err)
	}
}

func TestTrailAPI_Errors(t *testing.T) {
	send, userID := newTrailAPI(t)

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
		wantCode   string
	}{
		{"unknown owner", nethttp.MethodPost, "/api/v1/trails", createTrailBody, nethttp.StatusBadRequest, "VALIDATION_ERROR"},
		{"longitude out of range", nethttp.MethodPost, "/api/v1/trails", strings.NewReplacer(sampleUserID, userID, "6.1294", "200").Replace(createTrailBody), nethttp.StatusBadRequest, "VALIDATION_ERROR"},
		{"unknown trail", nethttp.MethodGet, "/api/v1/trails/00000000-0000-0000-0000-000000000000", "", nethttp.StatusNotFound, "NOT_FOUND"},
		{"malformed trail id", nethttp.MethodGet, "/api/v1/trails/not-a-uuid", "", nethttp.StatusBadRequest, "VALIDATION_ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec, env := send(tt.method, tt.path, tt.body)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (%s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if env.Error == nil || env.Error.Code != tt.wantCode {
				t.Fatalf("error = %+v, want %s", env.Error, tt.wantCode)
			}
			if strings.Contains(rec.Body.String(), "SQLSTATE") {
				t.Fatalf("database detail leaked: %s", rec.Body.String())
			}
		})
	}
}
