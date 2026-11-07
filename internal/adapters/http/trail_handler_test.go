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

	"github.com/traily-org/server/internal/domain/trail"
)

type fakeTrailService struct {
	getFn    func(ctx context.Context, id string) (trail.Trail, error)
	listFn   func(ctx context.Context, userID string) ([]trail.Trail, error)
	createFn func(ctx context.Context, t trail.Trail) (trail.Trail, error)
}

func (f *fakeTrailService) Get(ctx context.Context, id string) (trail.Trail, error) {
	return f.getFn(ctx, id)
}

func (f *fakeTrailService) ListByUser(ctx context.Context, userID string) ([]trail.Trail, error) {
	return f.listFn(ctx, userID)
}

func (f *fakeTrailService) Create(ctx context.Context, t trail.Trail) (trail.Trail, error) {
	return f.createFn(ctx, t)
}

const sampleUserID = "6f1c2c1e-3a53-4a52-9d6c-1d4d4f6a7b10"

var (
	ownerID     = sampleUserID
	description = "Boucle autour du lac"
	duration    = 5400
	sampleTrail = trail.Trail{
		ID:                       "trail-1",
		UserID:                   &ownerID,
		Name:                     "Tour du lac",
		Description:              &description,
		Difficulty:               trail.DifficultyEasy,
		DistanceMeters:           4200.5,
		ElevationGainMeters:      120,
		ElevationLossMeters:      118,
		EstimatedDurationSeconds: &duration,
		Path: []trail.Coordinate{
			{Longitude: 6.1294, Latitude: 45.8992},
			{Longitude: 6.1301, Latitude: 45.9004},
		},
		IsPublished: true,
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}
)

const createTrailBody = `{
	"user_id": "` + sampleUserID + `",
	"name": "Tour du lac",
	"description": "Boucle autour du lac",
	"difficulty": "easy",
	"distance_meters": 4200.5,
	"elevation_gain_meters": 120,
	"elevation_loss_meters": 118,
	"estimated_duration_seconds": 5400,
	"path": {"type": "LineString", "coordinates": [[6.1294, 45.8992], [6.1301, 45.9004]]}
}`

func serveTrail(t *testing.T, svc *fakeTrailService, method, path, body string) (*httptest.ResponseRecorder, envelope) {
	t.Helper()

	server := NewServer(NewUserHandler(&fakeUserService{}), NewTrailHandler(svc))
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

func echoCreate(received *trail.Trail) func(context.Context, trail.Trail) (trail.Trail, error) {
	return func(_ context.Context, tr trail.Trail) (trail.Trail, error) {
		*received = tr
		tr.ID = "trail-1"
		tr.CreatedAt = createdAt
		tr.UpdatedAt = createdAt
		return tr, nil
	}
}

func TestCreateTrail_Created(t *testing.T) {
	var received trail.Trail
	rec, env := serveTrail(t, &fakeTrailService{createFn: echoCreate(&received)}, nethttp.MethodPost, "/api/v1/trails", createTrailBody)

	if rec.Code != nethttp.StatusCreated {
		t.Fatalf("status = %d, want %d (%s)", rec.Code, nethttp.StatusCreated, rec.Body.String())
	}

	if received.UserID == nil || *received.UserID != sampleUserID {
		t.Fatalf("service received user = %v, want %q", received.UserID, sampleUserID)
	}
	if received.Name != "Tour du lac" || received.Difficulty != trail.DifficultyEasy {
		t.Fatalf("service received %+v", received)
	}
	if received.DistanceMeters != 4200.5 || received.ElevationGainMeters != 120 || received.ElevationLossMeters != 118 {
		t.Fatalf("service received metrics %+v", received)
	}
	wantPath := []trail.Coordinate{{Longitude: 6.1294, Latitude: 45.8992}, {Longitude: 6.1301, Latitude: 45.9004}}
	if len(received.Path) != 2 || received.Path[0] != wantPath[0] || received.Path[1] != wantPath[1] {
		t.Fatalf("service received path %v, want %v", received.Path, wantPath)
	}
	if !received.IsPublished {
		t.Fatal("a trail is published by default")
	}
	if received.IsOfficial {
		t.Fatal("a client cannot create an official trail")
	}

	var data map[string]any
	if err := json.Unmarshal(env.Data, &data); err != nil {
		t.Fatalf("data is not an object: %v", err)
	}
	if data["id"] != "trail-1" || data["user_id"] != sampleUserID || data["name"] != "Tour du lac" {
		t.Fatalf("unexpected data: %v", data)
	}
	path, _ := data["path"].(map[string]any)
	if path["type"] != "LineString" {
		t.Fatalf("path = %v, want a GeoJSON LineString", data["path"])
	}
	if data["created_at"] != "2026-10-29T10:00:00Z" {
		t.Fatalf("created_at = %v, want a UTC date", data["created_at"])
	}
	if _, leaked := data["password"]; leaked {
		t.Fatal("unexpected field in response")
	}
}

func TestCreateTrail_IgnoresOfficialFlag(t *testing.T) {
	var received trail.Trail
	body := strings.Replace(createTrailBody, `"difficulty": "easy",`, `"difficulty": "easy", "is_official": true,`, 1)

	rec, _ := serveTrail(t, &fakeTrailService{createFn: echoCreate(&received)}, nethttp.MethodPost, "/api/v1/trails", body)

	if rec.Code != nethttp.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, nethttp.StatusCreated)
	}
	if received.IsOfficial {
		t.Fatal("is_official must not be settable through the API")
	}
}

func TestCreateTrail_UnpublishedAndZeroMetrics(t *testing.T) {
	var received trail.Trail
	body := `{
		"user_id": "` + sampleUserID + `",
		"name": "Plat",
		"distance_meters": 1000,
		"elevation_gain_meters": 0,
		"elevation_loss_meters": 0,
		"is_published": false,
		"path": {"type": "LineString", "coordinates": [[1, 2, 350], [3, 4, 360]]}
	}`

	rec, _ := serveTrail(t, &fakeTrailService{createFn: echoCreate(&received)}, nethttp.MethodPost, "/api/v1/trails", body)

	if rec.Code != nethttp.StatusCreated {
		t.Fatalf("status = %d, want %d (%s)", rec.Code, nethttp.StatusCreated, rec.Body.String())
	}
	if received.IsPublished {
		t.Fatal("is_published=false must be honoured")
	}
	if received.Difficulty != "" {
		t.Fatalf("difficulty = %q, want it left to the domain default", received.Difficulty)
	}
	if received.Path[0] != (trail.Coordinate{Longitude: 1, Latitude: 2}) {
		t.Fatalf("path[0] = %v, want altitude dropped", received.Path[0])
	}
}

func TestCreateTrail_ValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"empty body", ``, "request body is required"},
		{"invalid JSON", `{`, "request body is not valid JSON"},
		{"missing user", strings.Replace(createTrailBody, `"user_id": "`+sampleUserID+`",`, ``, 1), "user_id: is required"},
		{"user not a UUID", strings.Replace(createTrailBody, sampleUserID, "bob", 1), "user_id: must be a valid UUID"},
		{"missing name", strings.Replace(createTrailBody, `"name": "Tour du lac",`, ``, 1), "name: is required"},
		{"unknown difficulty", strings.Replace(createTrailBody, `"easy"`, `"impossible"`, 1), "difficulty: must be one of"},
		{"missing distance", strings.Replace(createTrailBody, `"distance_meters": 4200.5,`, ``, 1), "distance_meters: is required"},
		{"negative distance", strings.Replace(createTrailBody, `4200.5`, `-1`, 1), "distance_meters: must be greater than or equal to 0"},
		{"negative elevation gain", strings.Replace(createTrailBody, `"elevation_gain_meters": 120`, `"elevation_gain_meters": -5`, 1), "elevation_gain_meters: must be greater than or equal to 0"},
		{"negative duration", strings.Replace(createTrailBody, `5400`, `-1`, 1), "estimated_duration_seconds: must be greater than or equal to 0"},
		{"missing path", `{"user_id": "` + sampleUserID + `", "name": "x", "distance_meters": 1, "elevation_gain_meters": 0, "elevation_loss_meters": 0}`, "path: is required"},
		{"path not a LineString", strings.Replace(createTrailBody, `"LineString"`, `"Point"`, 1), "type: is invalid"},
		{"path with one point", strings.Replace(createTrailBody, `[[6.1294, 45.8992], [6.1301, 45.9004]]`, `[[6.1294, 45.8992]]`, 1), "coordinates: must be at least 2"},
		{"point with one value", strings.Replace(createTrailBody, `[6.1301, 45.9004]`, `[6.1301]`, 1), "must be at least 2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			svc := &fakeTrailService{createFn: func(context.Context, trail.Trail) (trail.Trail, error) {
				called = true
				return trail.Trail{}, nil
			}}

			rec, env := serveTrail(t, svc, nethttp.MethodPost, "/api/v1/trails", tt.body)

			if rec.Code != nethttp.StatusBadRequest {
				t.Fatalf("status = %d, want %d (%s)", rec.Code, nethttp.StatusBadRequest, rec.Body.String())
			}
			if env.Error == nil || env.Error.Code != "VALIDATION_ERROR" {
				t.Fatalf("error = %+v, want VALIDATION_ERROR", env.Error)
			}
			if !strings.Contains(env.Error.Message, tt.want) {
				t.Fatalf("message = %q, want it to contain %q", env.Error.Message, tt.want)
			}
			if called {
				t.Fatal("the service must not be called on an invalid payload")
			}
		})
	}
}

func TestCreateTrail_ServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
		wantMsg    string
	}{
		{"domain validation", trail.ErrInvalidCoordinate, nethttp.StatusBadRequest, "VALIDATION_ERROR", trail.ErrInvalidCoordinate.Message},
		{"unexpected failure", errDatabase, nethttp.StatusInternalServerError, "INTERNAL_ERROR", "internal server error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeTrailService{createFn: func(context.Context, trail.Trail) (trail.Trail, error) {
				return trail.Trail{}, tt.err
			}}

			rec, env := serveTrail(t, svc, nethttp.MethodPost, "/api/v1/trails", createTrailBody)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if env.Error == nil || env.Error.Code != tt.wantCode || env.Error.Message != tt.wantMsg {
				t.Fatalf("error = %+v, want %s %q", env.Error, tt.wantCode, tt.wantMsg)
			}
			if strings.Contains(rec.Body.String(), "SQLSTATE") {
				t.Fatalf("database detail leaked: %s", rec.Body.String())
			}
		})
	}
}

func TestGetTrail(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		var gotID string
		svc := &fakeTrailService{getFn: func(_ context.Context, id string) (trail.Trail, error) {
			gotID = id
			return sampleTrail, nil
		}}

		rec, env := serveTrail(t, svc, nethttp.MethodGet, "/api/v1/trails/trail-1", "")

		if rec.Code != nethttp.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, nethttp.StatusOK)
		}
		if gotID != "trail-1" {
			t.Fatalf("service received id %q", gotID)
		}
		var data trailResponse
		if err := json.Unmarshal(env.Data, &data); err != nil {
			t.Fatalf("data is not a trail: %v", err)
		}
		if data.ID != "trail-1" || data.Difficulty != "easy" || len(data.Path.Coordinates) != 2 || data.Path.Coordinates[1][1] != 45.9004 {
			t.Fatalf("unexpected trail: %+v", data)
		}
		if data.CreatedAt.Location() != time.UTC {
			t.Fatalf("created_at location = %v, want UTC", data.CreatedAt.Location())
		}
	})

	t.Run("not found", func(t *testing.T) {
		svc := &fakeTrailService{getFn: func(context.Context, string) (trail.Trail, error) {
			return trail.Trail{}, trail.ErrNotFound
		}}

		rec, env := serveTrail(t, svc, nethttp.MethodGet, "/api/v1/trails/missing", "")

		if rec.Code != nethttp.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, nethttp.StatusNotFound)
		}
		if env.Error == nil || env.Error.Code != "NOT_FOUND" {
			t.Fatalf("error = %+v, want NOT_FOUND", env.Error)
		}
	})
}

func TestListTrails(t *testing.T) {
	t.Run("returns the trails of the user", func(t *testing.T) {
		var gotUserID string
		svc := &fakeTrailService{listFn: func(_ context.Context, userID string) ([]trail.Trail, error) {
			gotUserID = userID
			return []trail.Trail{sampleTrail, sampleTrail}, nil
		}}

		rec, env := serveTrail(t, svc, nethttp.MethodGet, "/api/v1/trails?user_id="+sampleUserID, "")

		if rec.Code != nethttp.StatusOK {
			t.Fatalf("status = %d, want %d (%s)", rec.Code, nethttp.StatusOK, rec.Body.String())
		}
		if gotUserID != sampleUserID {
			t.Fatalf("service received user %q, want %q", gotUserID, sampleUserID)
		}
		var data []trailResponse
		if err := json.Unmarshal(env.Data, &data); err != nil || len(data) != 2 {
			t.Fatalf("data = %s, want 2 trails (err %v)", env.Data, err)
		}
	})

	t.Run("no trail is an empty array", func(t *testing.T) {
		svc := &fakeTrailService{listFn: func(context.Context, string) ([]trail.Trail, error) {
			return nil, nil
		}}

		rec, env := serveTrail(t, svc, nethttp.MethodGet, "/api/v1/trails?user_id="+sampleUserID, "")

		if rec.Code != nethttp.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, nethttp.StatusOK)
		}
		if string(env.Data) != "[]" {
			t.Fatalf("data = %s, want []", env.Data)
		}
	})

	for name, path := range map[string]string{
		"missing user_id":     "/api/v1/trails",
		"user_id not a UUID":  "/api/v1/trails?user_id=bob",
		"user_id left blank ": "/api/v1/trails?user_id=",
	} {
		t.Run(name, func(t *testing.T) {
			svc := &fakeTrailService{listFn: func(context.Context, string) ([]trail.Trail, error) {
				t.Fatal("the service must not be called")
				return nil, errors.New("unreachable")
			}}

			rec, env := serveTrail(t, svc, nethttp.MethodGet, path, "")

			if rec.Code != nethttp.StatusBadRequest {
				t.Fatalf("status = %d, want %d", rec.Code, nethttp.StatusBadRequest)
			}
			if env.Error == nil || env.Error.Code != "VALIDATION_ERROR" || !strings.HasPrefix(env.Error.Message, "user_id:") {
				t.Fatalf("error = %+v, want a user_id VALIDATION_ERROR", env.Error)
			}
		})
	}
}
