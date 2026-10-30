package user

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// fakeRepository is an in-memory Repository that records the users it
// receives and echoes them back like a database would.
type fakeRepository struct {
	created    *User
	updated    *User
	stored     User
	err        error
	createCall int
	// emailLookup is the email received by the last GetByEmail call.
	emailLookup string
}

func (f *fakeRepository) GetByID(_ context.Context, _ string) (User, error) {
	if f.err != nil {
		return User{}, f.err
	}
	return f.stored, nil
}

func (f *fakeRepository) GetByEmail(_ context.Context, email string) (User, error) {
	f.emailLookup = email
	if f.err != nil {
		return User{}, f.err
	}
	return f.stored, nil
}

func (f *fakeRepository) Create(_ context.Context, u User) (User, error) {
	f.createCall++
	if f.err != nil {
		return User{}, f.err
	}
	received := u
	f.created = &received
	u.ID = "generated-id"
	return u, nil
}

func (f *fakeRepository) Update(_ context.Context, u User) (User, error) {
	if f.err != nil {
		return User{}, f.err
	}
	received := u
	f.updated = &received
	if u.Password == "" {
		u.Password = f.stored.Password
	}
	return u, nil
}

func (f *fakeRepository) Delete(_ context.Context, _ string) error {
	return f.err
}

func assertBcryptHashOf(t *testing.T, hash, plain string) {
	t.Helper()

	if hash == plain {
		t.Fatal("repository received the plain password")
	}
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		t.Fatalf("repository received an invalid bcrypt hash: %v", err)
	}
	if cost != BcryptCost {
		t.Fatalf("bcrypt cost = %d, want %d", cost, BcryptCost)
	}
	if !CheckPassword(hash, plain) {
		t.Fatal("stored hash does not match the plain password")
	}
}

func TestServiceCreate_WithPassword_StoresBcryptHash(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(repo)

	_, err := service.Create(context.Background(), User{Email: "alice@example.com", Password: "supersecret", Name: "Alice"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if repo.created == nil {
		t.Fatal("repository Create() was not called")
	}
	assertBcryptHashOf(t, repo.created.Password, "supersecret")
}

func TestServiceCreate_WithPassword_ReturnsUserWithoutPassword(t *testing.T) {
	service := NewService(&fakeRepository{})

	created, err := service.Create(context.Background(), User{Email: "alice@example.com", Password: "supersecret", Name: "Alice"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Password != "" {
		t.Fatalf("Create() Password = %q, want empty", created.Password)
	}
	if created.ID != "generated-id" || created.Email != "alice@example.com" {
		t.Fatalf("Create() = %+v, want repository fields", created)
	}
}

func TestServiceCreate_EmptyPassword_ReturnsErrPasswordRequired(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(repo)

	_, err := service.Create(context.Background(), User{Email: "alice@example.com", Name: "Alice"})
	if !errors.Is(err, ErrPasswordRequired) {
		t.Fatalf("Create() error = %v, want %v", err, ErrPasswordRequired)
	}
	if repo.createCall != 0 {
		t.Fatalf("repository Create() called %d times, want 0", repo.createCall)
	}
}

func TestServiceCreate_RepositoryError_ReturnsError(t *testing.T) {
	service := NewService(&fakeRepository{err: ErrEmailTaken})

	_, err := service.Create(context.Background(), User{Email: "alice@example.com", Password: "supersecret"})
	if !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("Create() error = %v, want %v", err, ErrEmailTaken)
	}
}

func TestServiceUpdate_WithPassword_StoresBcryptHash(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(repo)

	updated, err := service.Update(context.Background(), User{ID: "id", Email: "alice@example.com", Password: "newsecret", Name: "Alice"})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if repo.updated == nil {
		t.Fatal("repository Update() was not called")
	}
	assertBcryptHashOf(t, repo.updated.Password, "newsecret")
	if updated.Password != "" {
		t.Fatalf("Update() Password = %q, want empty", updated.Password)
	}
}

func TestServiceUpdate_EmptyPassword_PassesEmptyPasswordToRepository(t *testing.T) {
	repo := &fakeRepository{stored: User{Password: "$2a$12$existinghash"}}
	service := NewService(repo)

	updated, err := service.Update(context.Background(), User{ID: "id", Email: "alice@example.com", Name: "Alice"})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if repo.updated == nil {
		t.Fatal("repository Update() was not called")
	}
	if repo.updated.Password != "" {
		t.Fatalf("repository received Password = %q, want empty to keep the stored one", repo.updated.Password)
	}
	if updated.Password != "" {
		t.Fatalf("Update() Password = %q, want empty", updated.Password)
	}
}

func TestServiceUpdate_RepositoryError_ReturnsError(t *testing.T) {
	service := NewService(&fakeRepository{err: ErrNotFound})

	_, err := service.Update(context.Background(), User{ID: "id"})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Update() error = %v, want %v", err, ErrNotFound)
	}
}

func TestServiceGet_ExistingUser_ReturnsUserWithoutPassword(t *testing.T) {
	repo := &fakeRepository{stored: User{ID: "id", Email: "alice@example.com", Password: "$2a$12$existinghash"}}
	service := NewService(repo)

	got, err := service.Get(context.Background(), "id")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Password != "" {
		t.Fatalf("Get() Password = %q, want empty", got.Password)
	}
	if got.ID != "id" || got.Email != "alice@example.com" {
		t.Fatalf("Get() = %+v, want repository fields", got)
	}
}

func TestServiceGet_RepositoryError_ReturnsError(t *testing.T) {
	service := NewService(&fakeRepository{err: ErrNotFound})

	_, err := service.Get(context.Background(), "id")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() error = %v, want %v", err, ErrNotFound)
	}
}

func storedUserWithPassword(t *testing.T, plain string) User {
	t.Helper()

	hash, err := HashPassword(plain)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	return User{ID: "id", Email: "alice@example.com", Password: hash, Name: "Alice", Role: RoleUser}
}

func TestServiceAuthenticate_ValidCredentials_ReturnsUser(t *testing.T) {
	repo := &fakeRepository{stored: storedUserWithPassword(t, "supersecret")}
	service := NewService(repo)

	got, err := service.Authenticate(context.Background(), "Alice@Example.com", "supersecret")
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if got.ID != "id" || got.Email != "alice@example.com" || got.Role != RoleUser {
		t.Fatalf("Authenticate() = %+v, want repository fields", got)
	}
	if repo.emailLookup != "alice@example.com" {
		t.Fatalf("repository GetByEmail() received %q, want the normalized email", repo.emailLookup)
	}
}

func TestServiceAuthenticate_ValidCredentials_ReturnsUserWithoutPassword(t *testing.T) {
	service := NewService(&fakeRepository{stored: storedUserWithPassword(t, "supersecret")})

	got, err := service.Authenticate(context.Background(), "alice@example.com", "supersecret")
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if got.Password != "" {
		t.Fatalf("Authenticate() Password = %q, want empty", got.Password)
	}
}

func TestServiceAuthenticate_UnknownEmail_ReturnsErrInvalidCredentials(t *testing.T) {
	service := NewService(&fakeRepository{err: ErrNotFound})

	got, err := service.Authenticate(context.Background(), "nobody@example.com", "supersecret")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Authenticate() error = %v, want %v", err, ErrInvalidCredentials)
	}
	if got != (User{}) {
		t.Fatalf("Authenticate() = %+v, want zero user", got)
	}
}

func TestServiceAuthenticate_WrongPassword_ReturnsErrInvalidCredentials(t *testing.T) {
	service := NewService(&fakeRepository{stored: storedUserWithPassword(t, "supersecret")})

	got, err := service.Authenticate(context.Background(), "alice@example.com", "wrongpassword")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Authenticate() error = %v, want %v", err, ErrInvalidCredentials)
	}
	if got != (User{}) {
		t.Fatalf("Authenticate() = %+v, want zero user", got)
	}
}

func TestServiceAuthenticate_MigratedAccountWithoutPassword_ReturnsErrInvalidCredentials(t *testing.T) {
	service := NewService(&fakeRepository{stored: User{ID: "id", Email: "alice@example.com", Password: "!"}})

	for _, password := range []string{"!", "", "supersecret"} {
		if _, err := service.Authenticate(context.Background(), "alice@example.com", password); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("Authenticate(%q) error = %v, want %v", password, err, ErrInvalidCredentials)
		}
	}
}

func TestServiceAuthenticate_RepositoryError_ReturnsError(t *testing.T) {
	cause := errors.New("connection reset")
	service := NewService(&fakeRepository{err: cause})

	if _, err := service.Authenticate(context.Background(), "alice@example.com", "supersecret"); !errors.Is(err, cause) {
		t.Fatalf("Authenticate() error = %v, want %v", err, cause)
	}
}

func TestDummyPasswordHash_Value_IsBcryptHashWithProjectCost(t *testing.T) {
	cost, err := bcrypt.Cost([]byte(dummyPasswordHash))
	if err != nil {
		t.Fatalf("bcrypt.Cost(dummyPasswordHash) error = %v", err)
	}
	if cost != BcryptCost {
		t.Fatalf("dummy hash cost = %d, want %d so that unknown emails take as long as known ones", cost, BcryptCost)
	}
}

func TestNormalizeEmail_MixedCaseAndSpaces_ReturnsTrimmedLowercase(t *testing.T) {
	if got := NormalizeEmail("  Alice@Example.COM "); got != "alice@example.com" {
		t.Fatalf("NormalizeEmail() = %q, want %q", got, "alice@example.com")
	}
}

// fakeSessions records the users whose sessions were revoked.
type fakeSessions struct {
	revoked []string
	err     error
}

func (f *fakeSessions) RevokeAllSessions(_ context.Context, userID string) error {
	f.revoked = append(f.revoked, userID)
	return f.err
}

func TestServiceUpdateProfile_ProfileOnly_DoesNotNeedCurrentPassword(t *testing.T) {
	repo := &fakeRepository{stored: storedUserWithPassword(t, "supersecret")}
	sessions := &fakeSessions{}
	service := NewService(repo)
	service.SetSessionRevoker(sessions)

	got, err := service.UpdateProfile(context.Background(), User{ID: "id", Email: "ALICE@example.com", Name: "Alice B"}, "")
	if err != nil {
		t.Fatalf("UpdateProfile() error = %v", err)
	}
	if got.Name != "Alice B" || repo.updated == nil {
		t.Fatalf("UpdateProfile() = %+v, want the profile updated", got)
	}
	if len(sessions.revoked) != 0 {
		t.Fatalf("sessions revoked = %v, want none without a password change", sessions.revoked)
	}
}

func TestServiceUpdateProfile_CredentialChangeWithoutCurrentPassword_ReturnsErrCurrentPasswordRequired(t *testing.T) {
	tests := map[string]User{
		"new password": {ID: "id", Email: "alice@example.com", Name: "Alice", Password: "newsecret"},
		"new email":    {ID: "id", Email: "mallory@example.com", Name: "Alice"},
	}
	for name, u := range tests {
		t.Run(name, func(t *testing.T) {
			repo := &fakeRepository{stored: storedUserWithPassword(t, "supersecret")}

			if _, err := NewService(repo).UpdateProfile(context.Background(), u, ""); !errors.Is(err, ErrCurrentPasswordRequired) {
				t.Fatalf("UpdateProfile() error = %v, want %v", err, ErrCurrentPasswordRequired)
			}
			if repo.updated != nil {
				t.Fatal("repository Update() called without the current password")
			}
		})
	}
}

func TestServiceUpdateProfile_WrongCurrentPassword_ReturnsErrWrongCurrentPassword(t *testing.T) {
	repo := &fakeRepository{stored: storedUserWithPassword(t, "supersecret")}

	_, err := NewService(repo).UpdateProfile(context.Background(),
		User{ID: "id", Email: "mallory@example.com", Name: "Alice"}, "not-the-password")
	if !errors.Is(err, ErrWrongCurrentPassword) {
		t.Fatalf("UpdateProfile() error = %v, want %v", err, ErrWrongCurrentPassword)
	}
	if repo.updated != nil {
		t.Fatal("repository Update() called with a wrong current password")
	}
}

func TestServiceUpdateProfile_PasswordChanged_RevokesSessions(t *testing.T) {
	repo := &fakeRepository{stored: storedUserWithPassword(t, "supersecret")}
	sessions := &fakeSessions{}
	service := NewService(repo)
	service.SetSessionRevoker(sessions)

	if _, err := service.UpdateProfile(context.Background(),
		User{ID: "id", Email: "alice@example.com", Name: "Alice", Password: "newsecret"}, "supersecret"); err != nil {
		t.Fatalf("UpdateProfile() error = %v", err)
	}
	assertBcryptHashOf(t, repo.updated.Password, "newsecret")
	if len(sessions.revoked) != 1 || sessions.revoked[0] != "id" {
		t.Fatalf("sessions revoked = %v, want [id]", sessions.revoked)
	}
}

func TestServiceUpdateProfile_RevokeFails_ReturnsError(t *testing.T) {
	repo := &fakeRepository{stored: storedUserWithPassword(t, "supersecret")}
	boom := errors.New("boom")
	service := NewService(repo)
	service.SetSessionRevoker(&fakeSessions{err: boom})

	_, err := service.UpdateProfile(context.Background(),
		User{ID: "id", Email: "alice@example.com", Name: "Alice", Password: "newsecret"}, "supersecret")
	if !errors.Is(err, boom) {
		t.Fatalf("UpdateProfile() error = %v, want %v", err, boom)
	}
}
