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
}

func (f *fakeRepository) GetByID(_ context.Context, _ string) (User, error) {
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
