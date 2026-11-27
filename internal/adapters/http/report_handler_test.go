package http

import (
	"context"
	"encoding/json"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/traily-org/server/internal/adapters/http/middleware"
	"github.com/traily-org/server/internal/domain/report"
	"github.com/traily-org/server/internal/domain/user"
	"github.com/traily-org/server/internal/pkg/apperror"
)

const validTargetID = "11111111-1111-4111-8111-111111111111"

type fakeReportService struct {
	createFn func(ctx context.Context, in report.Input) (report.Report, error)
	listFn   func(ctx context.Context, f report.Filter) (report.Page, error)
}

func (f *fakeReportService) Create(ctx context.Context, in report.Input) (report.Report, error) {
	return f.createFn(ctx, in)
}

func (f *fakeReportService) List(ctx context.Context, flt report.Filter) (report.Page, error) {
	return f.listFn(ctx, flt)
}

func serveReports(t *testing.T, svc *fakeReportService, method, path, body, userID string, role user.Role) (*httptest.ResponseRecorder, envelope) {
	t.Helper()

	srv, err := NewServer(Deps{
		Users:            NewUserHandler(&fakeUserService{}),
		Auth:             noopAuthRoutes{},
		Reports:          NewReportHandler(svc),
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

func TestReportHandler_Create_Returns201WithReport(t *testing.T) {
	svc := &fakeReportService{createFn: func(_ context.Context, in report.Input) (report.Report, error) {
		if in.ReporterID != "u-1" || in.TargetType != report.TargetPhoto || in.TargetID != validTargetID || in.Reason != "nsfw" {
			t.Errorf("Create() input = %+v", in)
		}
		return report.Report{
			ID: "r-1", ReporterID: in.ReporterID, TargetType: in.TargetType, TargetID: in.TargetID,
			Reason: in.Reason, Status: report.StatusPending, CreatedAt: createdAt,
		}, nil
	}}

	rec, env := serveReports(t, svc, nethttp.MethodPost, "/api/v1/reports",
		`{"target_type":"photo","target_id":"`+validTargetID+`","reason":"nsfw"}`, "u-1", user.RoleUser)

	if rec.Code != nethttp.StatusCreated {
		t.Fatalf("status = %d, want 201 (body %s)", rec.Code, rec.Body.String())
	}
	got := decodeData(t, env)
	want := map[string]any{
		"id":          "r-1",
		"reporter_id": "u-1",
		"target_type": "photo",
		"target_id":   validTargetID,
		"reason":      "nsfw",
		"status":      "pending",
		"reviewed_by": nil,
		"reviewed_at": nil,
		"created_at":  "2026-10-29T10:00:00Z",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("data[%q] = %v, want %v", k, got[k], v)
		}
	}
}

func TestReportHandler_Create_InvalidPayload_Returns400(t *testing.T) {
	tests := map[string]string{
		"missing body":        ``,
		"unknown target type": `{"target_type":"comment","target_id":"` + validTargetID + `","reason":"spam"}`,
		"bad target id":       `{"target_type":"photo","target_id":"nope","reason":"spam"}`,
		"missing reason":      `{"target_type":"photo","target_id":"` + validTargetID + `"}`,
		"reason too long":     `{"target_type":"photo","target_id":"` + validTargetID + `","reason":"` + strings.Repeat("a", 1001) + `"}`,
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			svc := &fakeReportService{createFn: func(context.Context, report.Input) (report.Report, error) {
				t.Error("Create() must not be called")
				return report.Report{}, nil
			}}
			rec, env := serveReports(t, svc, nethttp.MethodPost, "/api/v1/reports", body, "u-1", user.RoleUser)
			if rec.Code != nethttp.StatusBadRequest || env.Error == nil || env.Error.Code != string(apperror.CodeValidation) {
				t.Errorf("status = %d, error = %+v, want 400 VALIDATION_ERROR", rec.Code, env.Error)
			}
		})
	}
}

func TestReportHandler_Create_RequiresAuthentication(t *testing.T) {
	svc := &fakeReportService{createFn: func(context.Context, report.Input) (report.Report, error) {
		t.Error("Create() must not be called")
		return report.Report{}, nil
	}}
	rec, _ := serveReports(t, svc, nethttp.MethodPost, "/api/v1/reports",
		`{"target_type":"photo","target_id":"`+validTargetID+`","reason":"spam"}`, "", "")
	if rec.Code != nethttp.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestReportHandler_Create_MapsDomainErrors(t *testing.T) {
	tests := []struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		{report.ErrTargetNotFound, nethttp.StatusNotFound, "NOT_FOUND"},
		{report.ErrCannotReportSelf, nethttp.StatusBadRequest, "VALIDATION_ERROR"},
		{report.ErrAlreadyReported, nethttp.StatusConflict, "CONFLICT"},
		{errDatabase, nethttp.StatusInternalServerError, "INTERNAL_ERROR"},
	}
	for _, tt := range tests {
		t.Run(tt.wantCode, func(t *testing.T) {
			svc := &fakeReportService{createFn: func(context.Context, report.Input) (report.Report, error) {
				return report.Report{}, tt.err
			}}
			rec, env := serveReports(t, svc, nethttp.MethodPost, "/api/v1/reports",
				`{"target_type":"user","target_id":"`+validTargetID+`","reason":"spam"}`, "u-1", user.RoleUser)
			if rec.Code != tt.wantStatus || env.Error == nil || env.Error.Code != tt.wantCode {
				t.Errorf("status = %d, error = %+v, want %d %s", rec.Code, env.Error, tt.wantStatus, tt.wantCode)
			}
			if strings.Contains(rec.Body.String(), "SQLSTATE") {
				t.Errorf("body leaks a database error: %s", rec.Body.String())
			}
		})
	}
}

func TestReportHandler_List_Moderator_ReturnsPage(t *testing.T) {
	svc := &fakeReportService{listFn: func(_ context.Context, f report.Filter) (report.Page, error) {
		if f.Status == nil || *f.Status != report.StatusPending {
			t.Errorf("Status = %v, want pending", f.Status)
		}
		if f.TargetType == nil || *f.TargetType != report.TargetPhoto {
			t.Errorf("TargetType = %v, want photo", f.TargetType)
		}
		if f.Limit != 10 || f.Offset != 20 {
			t.Errorf("Limit/Offset = %d/%d, want 10/20", f.Limit, f.Offset)
		}
		return report.Page{
			Items: []report.Report{{
				ID: "r-1", ReporterID: "u-1", TargetType: report.TargetPhoto, TargetID: validTargetID,
				Reason: "nsfw", Status: report.StatusPending, CreatedAt: createdAt,
			}},
			Total: 41, Limit: f.Limit, Offset: f.Offset,
		}, nil
	}}

	rec, env := serveReports(t, svc, nethttp.MethodGet,
		"/api/v1/moderation/reports?status=pending&target_type=photo&limit=10&offset=20", "", "mod-1", user.RoleModerator)

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
	got := decodeData(t, env)
	if got["total"] != float64(41) || got["limit"] != float64(10) || got["offset"] != float64(20) {
		t.Errorf("page = %v, want total 41, limit 10, offset 20", got)
	}
	items, _ := got["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("items = %v, want 1 item", got["items"])
	}
	item, _ := items[0].(map[string]any)
	for k, v := range map[string]any{"id": "r-1", "reporter_id": "u-1", "target_type": "photo", "target_id": validTargetID, "reason": "nsfw", "created_at": "2026-10-29T10:00:00Z"} {
		if item[k] != v {
			t.Errorf("item[%q] = %v, want %v", k, item[k], v)
		}
	}
}

func TestReportHandler_List_EmptyPage_ReturnsEmptyArray(t *testing.T) {
	svc := &fakeReportService{listFn: func(_ context.Context, f report.Filter) (report.Page, error) {
		return report.Page{Limit: 20}, nil
	}}
	rec, env := serveReports(t, svc, nethttp.MethodGet, "/api/v1/moderation/reports", "", "mod-1", user.RoleAdmin)
	if rec.Code != nethttp.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if items, ok := decodeData(t, env)["items"].([]any); !ok || len(items) != 0 {
		t.Errorf("items = %v, want []", decodeData(t, env)["items"])
	}
}

func TestReportHandler_List_RestrictedToModerators(t *testing.T) {
	svc := &fakeReportService{listFn: func(context.Context, report.Filter) (report.Page, error) {
		t.Error("List() must not be called")
		return report.Page{}, nil
	}}

	if rec, _ := serveReports(t, svc, nethttp.MethodGet, "/api/v1/moderation/reports", "", "u-1", user.RoleUser); rec.Code != nethttp.StatusForbidden {
		t.Errorf("user status = %d, want 403", rec.Code)
	}
	if rec, _ := serveReports(t, svc, nethttp.MethodGet, "/api/v1/moderation/reports", "", "", ""); rec.Code != nethttp.StatusUnauthorized {
		t.Errorf("anonymous status = %d, want 401", rec.Code)
	}
}

func TestReportHandler_List_InvalidQuery_Returns400(t *testing.T) {
	svc := &fakeReportService{listFn: func(context.Context, report.Filter) (report.Page, error) {
		t.Error("List() must not be called")
		return report.Page{}, nil
	}}
	for _, q := range []string{"status=open", "target_type=comment", "limit=0", "limit=101", "offset=-1", "limit=abc"} {
		t.Run(q, func(t *testing.T) {
			rec, _ := serveReports(t, svc, nethttp.MethodGet, "/api/v1/moderation/reports?"+q, "", "mod-1", user.RoleModerator)
			if rec.Code != nethttp.StatusBadRequest {
				t.Errorf("status = %d, want 400", rec.Code)
			}
		})
	}
}

func TestNewServer_ReportsWithoutGuards_Fails(t *testing.T) {
	_, err := NewServer(Deps{
		Users:   NewUserHandler(&fakeUserService{}),
		Auth:    noopAuthRoutes{},
		Reports: NewReportHandler(&fakeReportService{}),
	})
	if err == nil {
		t.Error("NewServer() error = nil, want an error: report routes need their guards")
	}
}
