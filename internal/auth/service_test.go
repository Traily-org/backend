package auth

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/traily-org/server/internal/domain/user"
	"github.com/traily-org/server/internal/pkg/apperror"
)

// fakeUsers is an in-memory UserStore.
type fakeUsers struct {
	byID    map[string]user.User
	authErr error
	nextID  int
}

func newFakeUsers() *fakeUsers {
	return &fakeUsers{byID: map[string]user.User{}}
}

func (f *fakeUsers) Create(_ context.Context, u user.User) (user.User, error) {
	f.nextID++
	u.ID = "user-" + strconv.Itoa(f.nextID)
	if u.Role == "" {
		u.Role = user.RoleUser
	}
	f.byID[u.ID] = u
	return u, nil
}

func (f *fakeUsers) Authenticate(_ context.Context, email, password string) (user.User, error) {
	if f.authErr != nil {
		return user.User{}, f.authErr
	}
	for _, u := range f.byID {
		if u.Email == email && u.Password == password {
			return u, nil
		}
	}
	return user.User{}, errors.New("bad credentials")
}

func (f *fakeUsers) Get(_ context.Context, id string) (user.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return user.User{}, user.ErrNotFound
	}
	return u, nil
}

// fakeTokens is an in-memory RefreshTokenRepository.
type fakeTokens struct {
	byID   map[string]*RefreshToken
	nextID int
}

func newFakeTokens() *fakeTokens {
	return &fakeTokens{byID: map[string]*RefreshToken{}}
}

func (f *fakeTokens) Create(_ context.Context, t RefreshToken) (RefreshToken, error) {
	f.nextID++
	t.ID = "token-" + strconv.Itoa(f.nextID)
	stored := t
	f.byID[t.ID] = &stored
	return t, nil
}

func (f *fakeTokens) GetByHash(_ context.Context, hash string) (RefreshToken, error) {
	for _, t := range f.byID {
		if t.TokenHash == hash {
			return *t, nil
		}
	}
	return RefreshToken{}, ErrRefreshTokenNotFound
}

func (f *fakeTokens) Revoke(_ context.Context, id string, at time.Time) (bool, error) {
	t, ok := f.byID[id]
	if !ok || t.RevokedAt != nil {
		return false, nil
	}
	t.RevokedAt = &at
	return true, nil
}

func (f *fakeTokens) RevokeAllForUser(_ context.Context, userID string, at time.Time) error {
	for _, t := range f.byID {
		if t.UserID == userID && t.RevokedAt == nil {
			t.RevokedAt = &at
		}
	}
	return nil
}

func (f *fakeTokens) activeForUser(userID string) int {
	n := 0
	for _, t := range f.byID {
		if t.UserID == userID && t.RevokedAt == nil {
			n++
		}
	}
	return n
}

// clock is a controllable time source.
type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

type fixture struct {
	svc    *Service
	users  *fakeUsers
	tokens *fakeTokens
	clock  *clock
}

func testConfig() Config {
	return Config{
		AccessSecret:  []byte("access-secret-for-tests-only-0123456789"),
		RefreshSecret: []byte("refresh-secret-for-tests-only-0123456789"),
		AccessTTL:     15 * time.Minute,
		RefreshTTL:    30 * 24 * time.Hour,
		Issuer:        "traily-test",
	}
}

func newFixture() fixture {
	users := newFakeUsers()
	tokens := newFakeTokens()
	c := &clock{t: time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)}
	svc := NewService(users, tokens, testConfig())
	svc.now = c.now
	return fixture{svc: svc, users: users, tokens: tokens, clock: c}
}

func (f fixture) register(t *testing.T, role user.Role) (user.User, TokenPair) {
	t.Helper()
	u, pair, err := f.svc.Register(context.Background(), user.User{
		Email:    "jane@example.com",
		Password: "secret",
		Name:     "Jane",
		Role:     role,
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	return u, pair
}

func signWith(t *testing.T, method jwt.SigningMethod, key any, claims jwt.Claims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return s
}

func TestRegister_ValidUser_ReturnsParsablePair(t *testing.T) {
	f := newFixture()

	u, pair := f.register(t, user.RoleModerator)

	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatalf("Register() pair = %+v, want both tokens set", pair)
	}
	if want := f.clock.t.Add(15 * time.Minute); !pair.AccessExpiresAt.Equal(want) {
		t.Errorf("AccessExpiresAt = %v, want %v", pair.AccessExpiresAt, want)
	}
	if want := f.clock.t.Add(30 * 24 * time.Hour); !pair.RefreshExpiresAt.Equal(want) {
		t.Errorf("RefreshExpiresAt = %v, want %v", pair.RefreshExpiresAt, want)
	}
	claims, err := f.svc.ParseAccessToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("ParseAccessToken() error = %v", err)
	}
	if claims.UserID != u.ID || claims.Role != user.RoleModerator {
		t.Errorf("claims = %+v, want UserID %q and role %q", claims, u.ID, user.RoleModerator)
	}
	if n := f.tokens.activeForUser(u.ID); n != 1 {
		t.Errorf("active refresh tokens = %d, want 1", n)
	}
}

func TestRegister_StoredRefreshToken_IsHashNotPlaintext(t *testing.T) {
	f := newFixture()

	_, pair := f.register(t, user.RoleUser)

	if len(f.tokens.byID) != 1 {
		t.Fatalf("stored tokens = %d, want 1", len(f.tokens.byID))
	}
	for _, stored := range f.tokens.byID {
		if stored.TokenHash == pair.RefreshToken || strings.Contains(stored.TokenHash, pair.RefreshToken) {
			t.Fatalf("stored hash contains the plaintext token")
		}
		if want := hashRefreshToken(testConfig().RefreshSecret, pair.RefreshToken); stored.TokenHash != want {
			t.Errorf("stored hash = %q, want HMAC %q", stored.TokenHash, want)
		}
		if len(stored.TokenHash) != 64 {
			t.Errorf("stored hash length = %d, want 64 hex chars", len(stored.TokenHash))
		}
	}
}

func TestLogin_ValidCredentials_ReturnsParsablePair(t *testing.T) {
	f := newFixture()
	registered, _ := f.register(t, user.RoleAdmin)

	u, pair, err := f.svc.Login(context.Background(), "jane@example.com", "secret")

	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if u.ID != registered.ID {
		t.Errorf("Login() user ID = %q, want %q", u.ID, registered.ID)
	}
	claims, err := f.svc.ParseAccessToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("ParseAccessToken() error = %v", err)
	}
	if claims.UserID != registered.ID || claims.Role != user.RoleAdmin {
		t.Errorf("claims = %+v, want UserID %q and role admin", claims, registered.ID)
	}
}

func TestLogin_StoreError_PropagatesErrorUnchanged(t *testing.T) {
	f := newFixture()
	wantErr := apperror.Unauthorized("invalid credentials")
	f.users.authErr = wantErr

	_, pair, err := f.svc.Login(context.Background(), "jane@example.com", "wrong")

	if !errors.Is(err, wantErr) {
		t.Fatalf("Login() error = %v, want %v", err, wantErr)
	}
	if pair != (TokenPair{}) {
		t.Errorf("Login() pair = %+v, want zero", pair)
	}
	if len(f.tokens.byID) != 0 {
		t.Errorf("stored tokens = %d, want 0", len(f.tokens.byID))
	}
}

func TestRefresh_ValidToken_RotatesToken(t *testing.T) {
	f := newFixture()
	u, first := f.register(t, user.RoleUser)
	f.clock.t = f.clock.t.Add(time.Hour)

	second, err := f.svc.Refresh(context.Background(), first.RefreshToken)

	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if second.RefreshToken == first.RefreshToken {
		t.Errorf("Refresh() returned the same refresh token")
	}
	if second.AccessToken == first.AccessToken {
		t.Errorf("Refresh() returned the same access token")
	}
	old, err := f.tokens.GetByHash(context.Background(), hashRefreshToken(testConfig().RefreshSecret, first.RefreshToken))
	if err != nil {
		t.Fatalf("GetByHash(old) error = %v", err)
	}
	if old.RevokedAt == nil || !old.RevokedAt.Equal(f.clock.t) {
		t.Errorf("old token RevokedAt = %v, want %v", old.RevokedAt, f.clock.t)
	}
	if n := f.tokens.activeForUser(u.ID); n != 1 {
		t.Errorf("active refresh tokens = %d, want 1", n)
	}
	claims, err := f.svc.ParseAccessToken(second.AccessToken)
	if err != nil || claims.UserID != u.ID || claims.Role != user.RoleUser {
		t.Errorf("ParseAccessToken() = %+v, %v; want user %q", claims, err, u.ID)
	}
}

func TestRefresh_RoleChanged_UsesCurrentRole(t *testing.T) {
	f := newFixture()
	u, first := f.register(t, user.RoleUser)
	u.Role = user.RoleModerator
	f.users.byID[u.ID] = u

	second, err := f.svc.Refresh(context.Background(), first.RefreshToken)

	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	claims, err := f.svc.ParseAccessToken(second.AccessToken)
	if err != nil || claims.Role != user.RoleModerator {
		t.Errorf("ParseAccessToken() = %+v, %v; want role moderator", claims, err)
	}
}

func TestRefresh_UnknownOrEmptyToken_ReturnsErrInvalidRefreshToken(t *testing.T) {
	f := newFixture()
	f.register(t, user.RoleUser)

	for _, token := range []string{"", "not-a-real-token"} {
		_, err := f.svc.Refresh(context.Background(), token)
		if !errors.Is(err, ErrInvalidRefreshToken) {
			t.Errorf("Refresh(%q) error = %v, want ErrInvalidRefreshToken", token, err)
		}
	}
}

func TestRefresh_ExpiredToken_ReturnsErrInvalidRefreshToken(t *testing.T) {
	f := newFixture()
	_, pair := f.register(t, user.RoleUser)
	f.clock.t = f.clock.t.Add(30 * 24 * time.Hour)

	_, err := f.svc.Refresh(context.Background(), pair.RefreshToken)

	if !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("Refresh() error = %v, want ErrInvalidRefreshToken", err)
	}
}

func TestRefresh_ReusedToken_RevokesAllUserTokens(t *testing.T) {
	f := newFixture()
	u, first := f.register(t, user.RoleUser)
	// A second, unrelated session of the same user.
	if _, _, err := f.svc.Login(context.Background(), "jane@example.com", "secret"); err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	second, err := f.svc.Refresh(context.Background(), first.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}

	_, err = f.svc.Refresh(context.Background(), first.RefreshToken)

	if !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("Refresh(reused) error = %v, want ErrInvalidRefreshToken", err)
	}
	if n := f.tokens.activeForUser(u.ID); n != 0 {
		t.Errorf("active refresh tokens = %d, want 0", n)
	}
	if _, err := f.svc.Refresh(context.Background(), second.RefreshToken); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Errorf("Refresh(rotated) error = %v, want ErrInvalidRefreshToken", err)
	}
}

// racingTokens simulates a concurrent refresh that revokes the token between
// GetByHash and Revoke.
type racingTokens struct{ *fakeTokens }

func (r racingTokens) Revoke(_ context.Context, _ string, _ time.Time) (bool, error) {
	return false, nil
}

func TestRefresh_ConcurrentRevoke_RevokesAllUserTokens(t *testing.T) {
	f := newFixture()
	u, pair := f.register(t, user.RoleUser)
	f.svc.tokens = racingTokens{f.tokens}

	_, err := f.svc.Refresh(context.Background(), pair.RefreshToken)

	if !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("Refresh() error = %v, want ErrInvalidRefreshToken", err)
	}
	if n := f.tokens.activeForUser(u.ID); n != 0 {
		t.Errorf("active refresh tokens = %d, want 0", n)
	}
}

func TestRefresh_DeletedUser_ReturnsErrInvalidRefreshToken(t *testing.T) {
	f := newFixture()
	u, pair := f.register(t, user.RoleUser)
	delete(f.users.byID, u.ID)

	_, err := f.svc.Refresh(context.Background(), pair.RefreshToken)

	if !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("Refresh() error = %v, want ErrInvalidRefreshToken", err)
	}
}

func TestLogout_ValidToken_RevokesItIdempotently(t *testing.T) {
	f := newFixture()
	u, pair := f.register(t, user.RoleUser)

	for i := range 2 {
		if err := f.svc.Logout(context.Background(), pair.RefreshToken); err != nil {
			t.Fatalf("Logout() call %d error = %v", i+1, err)
		}
	}

	if n := f.tokens.activeForUser(u.ID); n != 0 {
		t.Errorf("active refresh tokens = %d, want 0", n)
	}
	if _, err := f.svc.Refresh(context.Background(), pair.RefreshToken); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Errorf("Refresh() after logout error = %v, want ErrInvalidRefreshToken", err)
	}
}

func TestLogout_UnknownOrEmptyToken_ReturnsNil(t *testing.T) {
	f := newFixture()

	for _, token := range []string{"", "unknown"} {
		if err := f.svc.Logout(context.Background(), token); err != nil {
			t.Errorf("Logout(%q) error = %v, want nil", token, err)
		}
	}
}

func TestParseAccessToken_InvalidTokens_ReturnErrInvalidToken(t *testing.T) {
	f := newFixture()
	cfg := testConfig()
	now := f.clock.t
	valid := func() accessClaims {
		return accessClaims{
			Role: user.RoleUser,
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   "user-1",
				Issuer:    cfg.Issuer,
				IssuedAt:  jwt.NewNumericDate(now),
				ExpiresAt: jwt.NewNumericDate(now.Add(cfg.AccessTTL)),
				ID:        "jti",
			},
		}
	}
	with := func(edit func(*accessClaims)) accessClaims {
		c := valid()
		edit(&c)
		return c
	}

	_, pair := f.register(t, user.RoleUser)
	// Original header and signature around a payload escalated to admin.
	parts := strings.Split(pair.AccessToken, ".")
	forged := strings.Split(signWith(t, jwt.SigningMethodHS256, []byte("x"),
		with(func(c *accessClaims) { c.Role = user.RoleAdmin })), ".")
	tampered := parts[0] + "." + forged[1] + "." + parts[2]

	tests := []struct {
		name  string
		token string
	}{
		{"empty string", ""},
		{"garbage", "not.a.jwt"},
		{"wrong signature", signWith(t, jwt.SigningMethodHS256, []byte("another-secret"), valid())},
		{"tampered payload", tampered},
		{"alg none", signWith(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, valid())},
		{"alg HS512", signWith(t, jwt.SigningMethodHS512, cfg.AccessSecret, valid())},
		{"expired", signWith(t, jwt.SigningMethodHS256, cfg.AccessSecret,
			with(func(c *accessClaims) { c.ExpiresAt = jwt.NewNumericDate(now.Add(-time.Second)) }))},
		{"no expiry", signWith(t, jwt.SigningMethodHS256, cfg.AccessSecret,
			with(func(c *accessClaims) { c.ExpiresAt = nil }))},
		{"wrong issuer", signWith(t, jwt.SigningMethodHS256, cfg.AccessSecret,
			with(func(c *accessClaims) { c.Issuer = "someone-else" }))},
		{"missing subject", signWith(t, jwt.SigningMethodHS256, cfg.AccessSecret,
			with(func(c *accessClaims) { c.Subject = "" }))},
		{"unknown role", signWith(t, jwt.SigningMethodHS256, cfg.AccessSecret,
			with(func(c *accessClaims) { c.Role = "superuser" }))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := f.svc.ParseAccessToken(tt.token)
			if !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("ParseAccessToken() error = %v, want ErrInvalidToken", err)
			}
			if claims != (Claims{}) {
				t.Errorf("ParseAccessToken() claims = %+v, want zero", claims)
			}
		})
	}
}

func TestParseAccessToken_AfterTTL_ReturnsErrInvalidToken(t *testing.T) {
	f := newFixture()
	_, pair := f.register(t, user.RoleUser)
	f.clock.t = f.clock.t.Add(15*time.Minute + time.Second)

	if _, err := f.svc.ParseAccessToken(pair.AccessToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("ParseAccessToken() error = %v, want ErrInvalidToken", err)
	}
}

func TestNewRefreshToken_TwoCalls_ReturnDistinctURLSafeTokens(t *testing.T) {
	a, err := newRefreshToken()
	if err != nil {
		t.Fatalf("newRefreshToken() error = %v", err)
	}
	b, err := newRefreshToken()
	if err != nil {
		t.Fatalf("newRefreshToken() error = %v", err)
	}

	if a == b {
		t.Errorf("newRefreshToken() returned the same token twice")
	}
	// 32 bytes in unpadded base64url.
	if len(a) != 43 || strings.ContainsAny(a, "+/=") {
		t.Errorf("newRefreshToken() = %q, want 43 base64url chars without padding", a)
	}
}
