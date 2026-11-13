package ban

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/traily-org/server/internal/domain/user"
	"github.com/traily-org/server/internal/pkg/apperror"
)

// fakeRepo is an in-memory Repository.
type fakeRepo struct {
	reports     map[string]ReportRef
	bans        []Ban
	identifiers map[string]Identifier
	createErr   error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		identifiers: map[string]Identifier{},
		reports: map[string]ReportRef{
			"rep-1":         {ID: "rep-1", TargetUserID: strPtr("user-1")},
			"rep-mod2":      {ID: "rep-mod2", TargetUserID: strPtr("mod-2")},
			"rep-admin":     {ID: "rep-admin", TargetUserID: strPtr("admin-1")},
			"rep-reviewed":  {ID: "rep-reviewed", TargetUserID: strPtr("user-1")},
			"rep-dismissed": {ID: "rep-dismissed", TargetUserID: strPtr("user-1"), Dismissed: true},
			"rep-gone":      {ID: "rep-gone"},
		},
	}
}

func strPtr(s string) *string { return &s }

func (f *fakeRepo) Report(_ context.Context, id string) (ReportRef, error) {
	r, ok := f.reports[id]
	if !ok {
		return ReportRef{}, ErrReportNotFound
	}
	return r, nil
}

func (f *fakeRepo) Create(_ context.Context, b Ban, ids []Identifier) (Ban, error) {
	if f.createErr != nil {
		return Ban{}, f.createErr
	}
	b.ID = "ban-" + strconv.Itoa(len(f.bans)+1)
	f.bans = append(f.bans, b)
	for _, id := range ids {
		f.identifiers[string(id.Type)+":"+id.ValueHash] = id
	}
	return b, nil
}

func (f *fakeRepo) Active(_ context.Context, userID string, at time.Time) (Ban, error) {
	var best *Ban
	for i := range f.bans {
		b := &f.bans[i]
		if b.UserID != userID || !b.ActiveAt(at) {
			continue
		}
		if best == nil || b.ExpiresAt == nil || (best.ExpiresAt != nil && b.ExpiresAt.After(*best.ExpiresAt)) {
			best = b
		}
	}
	if best == nil {
		return Ban{}, ErrNotBanned
	}
	return *best, nil
}

func (f *fakeRepo) Lift(_ context.Context, userID, liftedBy string, at time.Time) (int, error) {
	n := 0
	for i := range f.bans {
		if f.bans[i].UserID == userID && f.bans[i].ActiveAt(at) {
			f.bans[i].LiftedAt = &at
			f.bans[i].LiftedBy = &liftedBy
			n++
		}
	}
	return n, nil
}

func (f *fakeRepo) IdentifierBanned(_ context.Context, kind IdentifierType, hash string, at time.Time) (bool, error) {
	id, ok := f.identifiers[string(kind)+":"+hash]
	return ok && (id.ExpiresAt == nil || at.Before(*id.ExpiresAt)), nil
}

// fakeUsers is an in-memory UserGetter.
type fakeUsers map[string]user.User

func (f fakeUsers) Get(_ context.Context, id string) (user.User, error) {
	u, ok := f[id]
	if !ok {
		return user.User{}, user.ErrNotFound
	}
	return u, nil
}

// fakeSessions records the users whose sessions were revoked.
type fakeSessions struct{ revoked []string }

func (f *fakeSessions) RevokeAllSessions(_ context.Context, userID string) error {
	f.revoked = append(f.revoked, userID)
	return nil
}

var (
	now       = time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	modAlice  = Moderator{ID: "mod-1", Role: user.RoleModerator}
	adminBob  = Moderator{ID: "admin-1", Role: user.RoleAdmin}
	testUsers = fakeUsers{
		"user-1":  {ID: "user-1", Email: "jane.doe@gmail.com", Role: user.RoleUser},
		"mod-2":   {ID: "mod-2", Email: "mod@example.com", Role: user.RoleModerator},
		"mod-1":   {ID: "mod-1", Email: "alice@example.com", Role: user.RoleModerator},
		"admin-1": {ID: "admin-1", Email: "bob@example.com", Role: user.RoleAdmin},
	}
)

type fixture struct {
	svc      *Service
	repo     *fakeRepo
	sessions *fakeSessions
}

func newFixture() fixture {
	repo := newFakeRepo()
	sessions := &fakeSessions{}
	svc := NewService(repo, testUsers)
	svc.SetSessionRevoker(sessions)
	svc.now = func() time.Time { return now }
	return fixture{svc: svc, repo: repo, sessions: sessions}
}

func at(d time.Duration) *time.Time {
	t := now.Add(d)
	return &t
}

func TestBan_TemporaryBan_StoresBanAndEmailAndRevokesSessions(t *testing.T) {
	f := newFixture()
	expires := at(7 * 24 * time.Hour)

	b, err := f.svc.Ban(context.Background(), Input{UserID: "user-1", ReportID: "rep-1", Reason: "  spam  ", ExpiresAt: expires, By: modAlice})

	if err != nil {
		t.Fatalf("Ban() error = %v", err)
	}
	if b.ReportID == nil || *b.ReportID != "rep-1" {
		t.Errorf("ReportID = %v, want rep-1", b.ReportID)
	}
	if b.UserID != "user-1" || b.Reason != "spam" || b.BannedBy == nil || *b.BannedBy != "mod-1" {
		t.Errorf("Ban() = %+v, want user-1 banned by mod-1 for \"spam\"", b)
	}
	if b.ExpiresAt == nil || !b.ExpiresAt.Equal(*expires) {
		t.Errorf("ExpiresAt = %v, want %v", b.ExpiresAt, expires)
	}
	id, ok := f.repo.identifiers["email:"+HashEmail("janedoe@gmail.com")]
	if !ok {
		t.Fatalf("identifiers = %v, want the hashed normalized email", f.repo.identifiers)
	}
	if id.ExpiresAt == nil || !id.ExpiresAt.Equal(*expires) {
		t.Errorf("identifier ExpiresAt = %v, want %v", id.ExpiresAt, expires)
	}
	if len(f.sessions.revoked) != 1 || f.sessions.revoked[0] != "user-1" {
		t.Errorf("revoked sessions = %v, want [user-1]", f.sessions.revoked)
	}
}

func TestBan_InvalidInput_ReturnsErrorWithoutStoring(t *testing.T) {
	tests := []struct {
		name string
		in   Input
		want error
	}{
		{"no report", Input{UserID: "user-1", Reason: "x", By: modAlice}, ErrReportRequired},
		{"unknown report", Input{UserID: "user-1", ReportID: "rep-404", Reason: "x", By: modAlice}, ErrReportNotFound},
		{"report about another user", Input{UserID: "user-1", ReportID: "rep-mod2", Reason: "x", By: adminBob}, ErrReportNotAboutUser},
		{"report target deleted", Input{UserID: "user-1", ReportID: "rep-gone", Reason: "x", By: modAlice}, ErrReportNotAboutUser},
		{"dismissed report", Input{UserID: "user-1", ReportID: "rep-dismissed", Reason: "x", By: modAlice}, ErrReportDismissed},
		{"empty reason", Input{UserID: "user-1", ReportID: "rep-1", Reason: "   ", By: modAlice}, ErrReasonRequired},
		{"self ban", Input{UserID: "mod-1", ReportID: "rep-1", Reason: "x", By: modAlice}, ErrCannotBanSelf},
		{"expiry now", Input{UserID: "user-1", ReportID: "rep-1", Reason: "x", ExpiresAt: at(0), By: modAlice}, ErrExpiryNotInFuture},
		{"expiry in the past", Input{UserID: "user-1", ReportID: "rep-1", Reason: "x", ExpiresAt: at(-time.Hour), By: modAlice}, ErrExpiryNotInFuture},
		{"unknown user", Input{UserID: "nobody", ReportID: "rep-1", Reason: "x", By: modAlice}, user.ErrNotFound},
		{"moderator bans moderator", Input{UserID: "mod-2", ReportID: "rep-mod2", Reason: "x", By: modAlice}, ErrCannotBanStaff},
		{"moderator bans admin", Input{UserID: "admin-1", ReportID: "rep-admin", Reason: "x", By: modAlice}, ErrCannotBanStaff},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture()

			_, err := f.svc.Ban(context.Background(), tt.in)

			if !errors.Is(err, tt.want) {
				t.Fatalf("Ban() error = %v, want %v", err, tt.want)
			}
			if len(f.repo.bans) != 0 || len(f.sessions.revoked) != 0 {
				t.Errorf("stored bans = %d, revoked = %v, want nothing", len(f.repo.bans), f.sessions.revoked)
			}
		})
	}
}

func TestBan_AdminBansModerator_Succeeds(t *testing.T) {
	f := newFixture()

	if _, err := f.svc.Ban(context.Background(), Input{UserID: "mod-2", ReportID: "rep-mod2", Reason: "abuse", By: adminBob}); err != nil {
		t.Fatalf("Ban() error = %v", err)
	}
	if len(f.repo.bans) != 1 || f.repo.bans[0].ExpiresAt != nil {
		t.Errorf("bans = %+v, want one permanent ban", f.repo.bans)
	}
}

func TestBan_RepositoryError_DoesNotRevokeSessions(t *testing.T) {
	f := newFixture()
	f.repo.createErr = apperror.Internal(errors.New("db down"))

	_, err := f.svc.Ban(context.Background(), Input{UserID: "user-1", ReportID: "rep-1", Reason: "x", By: modAlice})

	if !errors.Is(err, f.repo.createErr) {
		t.Fatalf("Ban() error = %v, want %v", err, f.repo.createErr)
	}
	if len(f.sessions.revoked) != 0 {
		t.Errorf("revoked sessions = %v, want none", f.sessions.revoked)
	}
}

func TestCheckUser_ActiveBan_ReturnsAccountBannedWithEndDate(t *testing.T) {
	f := newFixture()
	if _, err := f.svc.Ban(context.Background(), Input{UserID: "user-1", ReportID: "rep-1", Reason: "x", ExpiresAt: at(48 * time.Hour), By: modAlice}); err != nil {
		t.Fatalf("Ban() error = %v", err)
	}

	err := f.svc.CheckUser(context.Background(), "user-1")

	var appErr *apperror.Error
	if !errors.As(err, &appErr) || appErr.Code != apperror.CodeAccountBanned {
		t.Fatalf("CheckUser() error = %v, want an ACCOUNT_BANNED error", err)
	}
	if want := "account banned until 2026-11-15T12:00:00Z"; appErr.Message != want {
		t.Errorf("message = %q, want %q", appErr.Message, want)
	}
	if !errors.Is(err, ErrBanned) {
		t.Error("CheckUser() error must wrap ErrBanned")
	}
}

func TestCheckUser_PermanentAndTemporaryBans_ReportsPermanent(t *testing.T) {
	f := newFixture()
	for _, exp := range []*time.Time{at(time.Hour), nil} {
		if _, err := f.svc.Ban(context.Background(), Input{UserID: "user-1", ReportID: "rep-1", Reason: "x", ExpiresAt: exp, By: modAlice}); err != nil {
			t.Fatalf("Ban() error = %v", err)
		}
	}

	err := f.svc.CheckUser(context.Background(), "user-1")

	var appErr *apperror.Error
	if !errors.As(err, &appErr) || appErr.Message != "account permanently banned" {
		t.Fatalf("CheckUser() error = %v, want \"account permanently banned\"", err)
	}
}

func TestCheckUser_ExpiredOrLiftedOrNoBan_ReturnsNil(t *testing.T) {
	f := newFixture()
	if _, err := f.svc.Ban(context.Background(), Input{UserID: "user-1", ReportID: "rep-1", Reason: "x", ExpiresAt: at(time.Hour), By: modAlice}); err != nil {
		t.Fatalf("Ban() error = %v", err)
	}

	if err := f.svc.CheckUser(context.Background(), "mod-2"); err != nil {
		t.Errorf("CheckUser(not banned) error = %v, want nil", err)
	}

	f.svc.now = func() time.Time { return now.Add(time.Hour) }
	if err := f.svc.CheckUser(context.Background(), "user-1"); err != nil {
		t.Errorf("CheckUser(after expiry) error = %v, want nil", err)
	}
}

func TestLift_ActiveBan_EndsItAndSecondCallReturnsErrNotBanned(t *testing.T) {
	f := newFixture()
	if _, err := f.svc.Ban(context.Background(), Input{UserID: "user-1", ReportID: "rep-1", Reason: "x", By: modAlice}); err != nil {
		t.Fatalf("Ban() error = %v", err)
	}

	if err := f.svc.Lift(context.Background(), "user-1", adminBob); err != nil {
		t.Fatalf("Lift() error = %v", err)
	}
	if err := f.svc.CheckUser(context.Background(), "user-1"); err != nil {
		t.Errorf("CheckUser() after Lift error = %v, want nil", err)
	}
	if got := f.repo.bans[0].LiftedBy; got == nil || *got != "admin-1" {
		t.Errorf("LiftedBy = %v, want admin-1", got)
	}
	if err := f.svc.Lift(context.Background(), "user-1", adminBob); !errors.Is(err, ErrNotBanned) {
		t.Errorf("second Lift() error = %v, want ErrNotBanned", err)
	}
}

func TestCheckEmail_BannedAddressUnderAnotherSpelling_ReturnsAccountBanned(t *testing.T) {
	f := newFixture()
	if _, err := f.svc.Ban(context.Background(), Input{UserID: "user-1", ReportID: "rep-1", Reason: "x", ExpiresAt: at(time.Hour), By: modAlice}); err != nil {
		t.Fatalf("Ban() error = %v", err)
	}

	err := f.svc.CheckEmail(context.Background(), "JaneDoe+new@googlemail.com")

	var appErr *apperror.Error
	if !errors.As(err, &appErr) || appErr.Code != apperror.CodeAccountBanned || !errors.Is(err, ErrBanned) {
		t.Fatalf("CheckEmail() error = %v, want an ACCOUNT_BANNED error wrapping ErrBanned", err)
	}

	if err := f.svc.CheckEmail(context.Background(), "someone@gmail.com"); err != nil {
		t.Errorf("CheckEmail(other address) error = %v, want nil", err)
	}
	f.svc.now = func() time.Time { return now.Add(time.Hour) }
	if err := f.svc.CheckEmail(context.Background(), "jane.doe@gmail.com"); err != nil {
		t.Errorf("CheckEmail(after expiry) error = %v, want nil", err)
	}
}

func TestBan_ReviewedReport_Succeeds(t *testing.T) {
	f := newFixture()

	if _, err := f.svc.Ban(context.Background(), Input{UserID: "user-1", ReportID: "rep-reviewed", Reason: "x", By: modAlice}); err != nil {
		t.Fatalf("Ban() error = %v", err)
	}
}
