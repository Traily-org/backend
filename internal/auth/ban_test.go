package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/traily-org/server/internal/domain/user"
	"github.com/traily-org/server/internal/pkg/apperror"
)

// errTestBanned stands for the error returned by a ban checker.
var errTestBanned = &apperror.Error{Code: apperror.CodeAccountBanned, Message: "account permanently banned"}

// fakeBans bans the users and emails it lists.
type fakeBans struct {
	users  map[string]bool
	emails map[string]bool
}

func (f *fakeBans) CheckUser(_ context.Context, userID string) error {
	if f.users[userID] {
		return errTestBanned
	}
	return nil
}

func (f *fakeBans) CheckEmail(_ context.Context, email string) error {
	if f.emails[email] {
		return errTestBanned
	}
	return nil
}

func withBans(f fixture) *fakeBans {
	bans := &fakeBans{users: map[string]bool{}, emails: map[string]bool{}}
	f.svc.SetBanChecker(bans)
	return bans
}

func TestRegister_BannedEmail_ReturnsBanErrorWithoutCreatingUser(t *testing.T) {
	f := newFixture()
	bans := withBans(f)
	bans.emails["jane@example.com"] = true

	_, pair, err := f.svc.Register(context.Background(), user.User{Email: "jane@example.com", Password: "secret", Name: "Jane"})

	if !errors.Is(err, errTestBanned) {
		t.Fatalf("Register() error = %v, want the ban error", err)
	}
	if pair != (TokenPair{}) || len(f.users.byID) != 0 || len(f.tokens.byID) != 0 {
		t.Errorf("Register() created users = %d, tokens = %d, want none", len(f.users.byID), len(f.tokens.byID))
	}
}

func TestLogin_BannedUser_ReturnsBanErrorWithoutIssuingTokens(t *testing.T) {
	f := newFixture()
	u, _ := f.register(t, user.RoleUser)
	bans := withBans(f)
	bans.users[u.ID] = true
	before := len(f.tokens.byID)

	_, pair, err := f.svc.Login(context.Background(), "jane@example.com", "secret")

	if !errors.Is(err, errTestBanned) {
		t.Fatalf("Login() error = %v, want the ban error", err)
	}
	if pair != (TokenPair{}) || len(f.tokens.byID) != before {
		t.Errorf("Login() issued tokens = %d, want none", len(f.tokens.byID)-before)
	}
}

func TestLogin_BannedUserWrongPassword_ReturnsCredentialsError(t *testing.T) {
	f := newFixture()
	u, _ := f.register(t, user.RoleUser)
	bans := withBans(f)
	bans.users[u.ID] = true

	_, _, err := f.svc.Login(context.Background(), "jane@example.com", "wrong")

	if err == nil || errors.Is(err, errTestBanned) {
		t.Fatalf("Login() error = %v, want a credentials error, not the ban", err)
	}
}

func TestRefresh_BannedUser_ReturnsBanErrorAndRevokesPresentedToken(t *testing.T) {
	f := newFixture()
	u, pair := f.register(t, user.RoleUser)
	bans := withBans(f)
	bans.users[u.ID] = true

	_, err := f.svc.Refresh(context.Background(), pair.RefreshToken)

	if !errors.Is(err, errTestBanned) {
		t.Fatalf("Refresh() error = %v, want the ban error", err)
	}
	if n := f.tokens.activeForUser(u.ID); n != 0 {
		t.Errorf("active refresh tokens = %d, want 0", n)
	}
}

func TestLogin_NotBannedUser_Succeeds(t *testing.T) {
	f := newFixture()
	f.register(t, user.RoleUser)
	withBans(f)

	if _, _, err := f.svc.Login(context.Background(), "jane@example.com", "secret"); err != nil {
		t.Fatalf("Login() error = %v", err)
	}
}
