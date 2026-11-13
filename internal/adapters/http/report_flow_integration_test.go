//go:build integration

package http_test

import (
	"bytes"
	"context"
	"encoding/json"
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
	"github.com/traily-org/server/internal/domain/report"
	"github.com/traily-org/server/internal/domain/user"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/internal/testutil"
)

type reportFlow struct {
	t    *testing.T
	h    nethttp.Handler
	pool *pgxpool.Pool
}

// newReportFlow wires the real repositories, services, middlewares and router
// on the shared test database, as cmd/main.go does.
func newReportFlow(t *testing.T) *reportFlow {
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
	bans := ban.NewService(postgres.NewBanRepository(pool), users)
	authService.SetBanChecker(bans)

	srv, err := http.NewServer(http.Deps{
		Users:            http.NewUserHandler(users),
		Auth:             http.NewAuthHandler(authService),
		Moderation:       http.NewModerationHandler(bans),
		Reports:          http.NewReportHandler(report.NewService(postgres.NewReportRepository(pool))),
		Authenticate:     middleware.Authenticate(authService),
		RejectBanned:     middleware.RejectBanned(bans),
		RequireModerator: middleware.RequireRole(user.RoleModerator, user.RoleAdmin),
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	return &reportFlow{t: t, h: srv.Handler(), pool: pool}
}

type rawEnvelope struct {
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Code string `json:"code"`
	} `json:"error"`
}

func (f *reportFlow) call(method, path, token, body string) (int, rawEnvelope) {
	f.t.Helper()

	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	f.h.ServeHTTP(rec, req)

	var env rawEnvelope
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
			f.t.Fatalf("%s %s: decode body %q: %v", method, path, rec.Body.String(), err)
		}
	}
	return rec.Code, env
}

// signUp registers an account and returns its ID and access token. A non-user
// role is set directly in the database, as there is no API to grant it.
func (f *reportFlow) signUp(name string, role user.Role) (id, token string) {
	f.t.Helper()

	status, env := f.call(nethttp.MethodPost, "/api/v1/auth/register", "",
		`{"email":"`+name+`@example.com","password":"supersecret","name":"`+name+`"}`)
	expectStatus(f.t, "register "+name, status, nethttp.StatusCreated)

	var data struct {
		User   struct{ ID string } `json:"user"`
		Tokens struct {
			AccessToken string `json:"access_token"`
		} `json:"tokens"`
	}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		f.t.Fatalf("decode register response: %v", err)
	}

	if role != user.RoleUser {
		if _, err := f.pool.Exec(context.Background(), `UPDATE users SET role = $1 WHERE id = $2`, string(role), data.User.ID); err != nil {
			f.t.Fatalf("set role: %v", err)
		}
		// The role is carried by the access token: sign in again to get it.
		status, env = f.call(nethttp.MethodPost, "/api/v1/auth/login", "",
			`{"email":"`+name+`@example.com","password":"supersecret"}`)
		expectStatus(f.t, "login "+name, status, nethttp.StatusOK)
		if err := json.Unmarshal(env.Data, &data); err != nil {
			f.t.Fatalf("decode login response: %v", err)
		}
	}
	return data.User.ID, data.Tokens.AccessToken
}

type reportPage struct {
	Items []struct {
		ID         string  `json:"id"`
		ReporterID string  `json:"reporter_id"`
		TargetType string  `json:"target_type"`
		TargetID   string  `json:"target_id"`
		Reason     string  `json:"reason"`
		Status     string  `json:"status"`
		ReviewedBy *string `json:"reviewed_by"`
		CreatedAt  string  `json:"created_at"`
	} `json:"items"`
	Total int `json:"total"`
}

func (f *reportFlow) list(token, query string) reportPage {
	f.t.Helper()

	status, env := f.call(nethttp.MethodGet, "/api/v1/moderation/reports"+query, token, "")
	expectStatus(f.t, "list reports", status, nethttp.StatusOK)

	var page reportPage
	if err := json.Unmarshal(env.Data, &page); err != nil {
		f.t.Fatalf("decode report page: %v", err)
	}
	return page
}

func TestReportFlow_HikerReports_ModeratorSeesItImmediately(t *testing.T) {
	f := newReportFlow(t)
	hikerID, hiker := f.signUp("hiker", user.RoleUser)
	authorID, _ := f.signUp("author", user.RoleUser)
	_, moderator := f.signUp("moderator", user.RoleModerator)

	var photoID string
	if err := f.pool.QueryRow(context.Background(),
		`INSERT INTO photos (user_id, storage_key) VALUES ($1, 'photos/1.jpg') RETURNING id::text`, authorID).Scan(&photoID); err != nil {
		t.Fatalf("create photo: %v", err)
	}

	before := time.Now().UTC().Add(-time.Second)
	status, env := f.call(nethttp.MethodPost, "/api/v1/reports", hiker,
		`{"target_type":"photo","target_id":"`+photoID+`","reason":"Contenu inapproprié"}`)
	expectStatus(t, "report a photo", status, nethttp.StatusCreated)
	if env.Error != nil {
		t.Fatalf("report a photo: error = %+v", env.Error)
	}

	page := f.list(moderator, "?status=pending")
	if page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("dashboard shows %d reports (total %d), want the new one", len(page.Items), page.Total)
	}
	got := page.Items[0]
	if got.ReporterID != hikerID || got.TargetType != "photo" || got.TargetID != photoID || got.Reason != "Contenu inapproprié" {
		t.Errorf("report = %+v, want author, target and reason traced", got)
	}
	if got.Status != "pending" || got.ReviewedBy != nil {
		t.Errorf("report = %+v, want pending and not reviewed", got)
	}
	created, err := time.Parse(time.RFC3339Nano, got.CreatedAt)
	if err != nil || created.Before(before) || time.Since(created) > time.Minute {
		t.Errorf("created_at = %q (%v), want within the last minute", got.CreatedAt, err)
	}
}

func TestReportFlow_Rules(t *testing.T) {
	f := newReportFlow(t)
	hikerID, hiker := f.signUp("hiker", user.RoleUser)
	otherID, _ := f.signUp("other", user.RoleUser)
	_, moderator := f.signUp("moderator", user.RoleModerator)

	reportUser := func(token, targetID string) (int, rawEnvelope) {
		return f.call(nethttp.MethodPost, "/api/v1/reports", token,
			`{"target_type":"user","target_id":"`+targetID+`","reason":"harcèlement"}`)
	}
	expectCode := func(step string, status int, env rawEnvelope, wantStatus int, wantCode string) {
		t.Helper()
		expectStatus(t, step, status, wantStatus)
		if env.Error == nil || env.Error.Code != wantCode {
			t.Fatalf("%s: error = %+v, want %s", step, env.Error, wantCode)
		}
	}

	status, env := reportUser("", otherID)
	expectCode("anonymous report", status, env, nethttp.StatusUnauthorized, "UNAUTHORIZED")

	status, env = reportUser(hiker, hikerID)
	expectCode("report yourself", status, env, nethttp.StatusBadRequest, "VALIDATION_ERROR")

	status, env = reportUser(hiker, "00000000-0000-4000-8000-000000000000")
	expectCode("report an unknown user", status, env, nethttp.StatusNotFound, "NOT_FOUND")

	status, env = f.call(nethttp.MethodPost, "/api/v1/reports", hiker, `{"target_type":"user","target_id":"nope","reason":"x"}`)
	expectCode("report with a bad id", status, env, nethttp.StatusBadRequest, "VALIDATION_ERROR")

	status, _ = reportUser(hiker, otherID)
	expectStatus(t, "report another user", status, nethttp.StatusCreated)
	status, env = reportUser(hiker, otherID)
	expectCode("report the same user twice", status, env, nethttp.StatusConflict, "CONFLICT")

	// Another hiker can still report the same user.
	status, _ = reportUser(f.mustToken("third"), otherID)
	expectStatus(t, "second reporter", status, nethttp.StatusCreated)

	status, env = f.call(nethttp.MethodGet, "/api/v1/moderation/reports", hiker, "")
	expectCode("hiker lists reports", status, env, nethttp.StatusForbidden, "FORBIDDEN")
	status, env = f.call(nethttp.MethodGet, "/api/v1/moderation/reports", "", "")
	expectCode("anonymous lists reports", status, env, nethttp.StatusUnauthorized, "UNAUTHORIZED")

	// Handled reports can be reported again, and filters see them.
	if _, err := f.pool.Exec(context.Background(), `UPDATE reports SET status = 'dismissed' WHERE reporter_id = $1`, hikerID); err != nil {
		t.Fatalf("dismiss report: %v", err)
	}
	status, _ = reportUser(hiker, otherID)
	expectStatus(t, "report again after dismissal", status, nethttp.StatusCreated)

	if got := f.list(moderator, "").Total; got != 3 {
		t.Errorf("all reports = %d, want 3", got)
	}
	if got := f.list(moderator, "?status=dismissed").Total; got != 1 {
		t.Errorf("dismissed reports = %d, want 1", got)
	}
	if got := f.list(moderator, "?status=pending&target_type=user&limit=1&offset=1"); got.Total != 2 || len(got.Items) != 1 {
		t.Errorf("page = %d items, total %d, want 1 item of 2", len(got.Items), got.Total)
	}
}

func (f *reportFlow) mustToken(name string) string {
	f.t.Helper()
	_, token := f.signUp(name, user.RoleUser)
	return token
}
