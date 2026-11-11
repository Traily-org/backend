//go:build integration

package user_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/user"
	"github.com/traily-org/server/internal/infrastructure/postgres"
	"github.com/traily-org/server/internal/testutil"
)

func newTestService(t *testing.T) *user.Service {
	t.Helper()

	pool := testutil.NewPool(t)
	repo := postgres.NewUserRepository(pool)
	return user.NewService(repo)
}

func TestService_CreateAndGet(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	created, err := service.Create(ctx, user.User{
		Email:    "alice@example.com",
		Password: "supersecret",
		Name:     "Alice",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID == "" {
		t.Fatal("Create() should return a generated ID")
	}

	fetched, err := service.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if fetched.Email != "alice@example.com" || fetched.Name != "Alice" {
		t.Fatalf("Get() = %+v, want matching fields", fetched)
	}
}

func TestService_Create_DuplicateEmail(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	u := user.User{Email: "bob@example.com", Password: "supersecret", Name: "Bob"}

	if _, err := service.Create(ctx, u); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}

	if _, err := service.Create(ctx, u); err != user.ErrEmailTaken {
		t.Fatalf("second Create() error = %v, want %v", err, user.ErrEmailTaken)
	}
}

func TestService_Update(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	created, err := service.Create(ctx, user.User{
		Email:    "carol@example.com",
		Password: "supersecret",
		Name:     "Carol",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	created.Name = "Carol Updated"
	updated, err := service.Update(ctx, created)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Name != "Carol Updated" {
		t.Fatalf("Update() Name = %q, want %q", updated.Name, "Carol Updated")
	}
}

func TestService_Update_NotFound(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	_, err := service.Update(ctx, user.User{ID: "00000000-0000-0000-0000-000000000000", Email: "nobody@example.com", Password: "x", Name: "Nobody"})
	if err != user.ErrNotFound {
		t.Fatalf("Update() error = %v, want %v", err, user.ErrNotFound)
	}
}

func TestService_Delete(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	created, err := service.Create(ctx, user.User{
		Email:    "dave@example.com",
		Password: "supersecret",
		Name:     "Dave",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := service.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if _, err := service.Get(ctx, created.ID); err != user.ErrNotFound {
		t.Fatalf("Get() after Delete() error = %v, want %v", err, user.ErrNotFound)
	}
}

func TestService_Delete_NotFound(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	err := service.Delete(ctx, "00000000-0000-0000-0000-000000000000")
	if err != user.ErrNotFound {
		t.Fatalf("Delete() error = %v, want %v", err, user.ErrNotFound)
	}
}

func TestService_Create_StoresBcryptHash(t *testing.T) {
	pool := testutil.NewPool(t)
	service := user.NewService(postgres.NewUserRepository(pool))
	ctx := context.Background()

	created, err := service.Create(ctx, user.User{Email: "erin@example.com", Password: "supersecret", Name: "Erin"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if stored := storedPassword(t, pool, created.ID); !user.CheckPassword(stored, "supersecret") {
		t.Fatalf("stored password %q is not a bcrypt hash of the plain password", stored)
	}
}

func TestService_Update_EmptyPassword_KeepsStoredHash(t *testing.T) {
	pool := testutil.NewPool(t)
	service := user.NewService(postgres.NewUserRepository(pool))
	ctx := context.Background()

	created, err := service.Create(ctx, user.User{Email: "frank@example.com", Password: "supersecret", Name: "Frank"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	before := storedPassword(t, pool, created.ID)

	created.Name = "Frank Updated"
	if _, err := service.Update(ctx, created); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if after := storedPassword(t, pool, created.ID); after != before {
		t.Fatalf("stored password changed on update without password: %q -> %q", before, after)
	}
}

func strPtr(s string) *string { return &s }

func TestService_Create_WithProfileFields_StoresAndReadsThem(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	created, err := service.Create(ctx, user.User{
		Email:       "gina@example.com",
		Password:    "supersecret",
		Name:        "Gina",
		Username:    strPtr("gina_42"),
		DisplayName: strPtr("Gina G."),
		AvatarURL:   strPtr("https://cdn.example.com/gina.png"),
		Role:        user.RoleAdmin,
		TotalPoints: 1000,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	fetched, err := service.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if fetched.Username == nil || *fetched.Username != "gina_42" {
		t.Errorf("Username = %v, want gina_42", fetched.Username)
	}
	if fetched.DisplayName == nil || *fetched.DisplayName != "Gina G." {
		t.Errorf("DisplayName = %v, want Gina G.", fetched.DisplayName)
	}
	if fetched.AvatarURL == nil || *fetched.AvatarURL != "https://cdn.example.com/gina.png" {
		t.Errorf("AvatarURL = %v, want https://cdn.example.com/gina.png", fetched.AvatarURL)
	}
	if fetched.Role != user.RoleUser {
		t.Errorf("Role = %q, want %q (role must not be writable)", fetched.Role, user.RoleUser)
	}
	if fetched.TotalPoints != 0 {
		t.Errorf("TotalPoints = %d, want 0 (points must not be writable)", fetched.TotalPoints)
	}
	if fetched.Password != "" {
		t.Error("Get() must not return the password hash")
	}
}

func TestService_Create_WithoutProfileFields_ReadsNil(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	created, err := service.Create(ctx, user.User{Email: "hank@example.com", Password: "supersecret", Name: "Hank"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Username != nil || created.DisplayName != nil || created.AvatarURL != nil {
		t.Fatalf("Create() = %+v, want nil profile fields", created)
	}
	if created.Role != user.RoleUser || created.TotalPoints != 0 {
		t.Fatalf("Create() role/points = %q/%d, want user/0", created.Role, created.TotalPoints)
	}
}

func TestService_Create_DuplicateUsername_ReturnsErrUsernameTaken(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	if _, err := service.Create(ctx, user.User{Email: "ivy@example.com", Password: "supersecret", Name: "Ivy", Username: strPtr("ivy")}); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}

	_, err := service.Create(ctx, user.User{Email: "ivy2@example.com", Password: "supersecret", Name: "Ivy 2", Username: strPtr("ivy")})
	if !errors.Is(err, user.ErrUsernameTaken) {
		t.Fatalf("second Create() error = %v, want %v", err, user.ErrUsernameTaken)
	}
}

func TestService_Update_DuplicateUsername_ReturnsErrUsernameTaken(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	if _, err := service.Create(ctx, user.User{Email: "jack@example.com", Password: "supersecret", Name: "Jack", Username: strPtr("jack")}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	other, err := service.Create(ctx, user.User{Email: "kate@example.com", Password: "supersecret", Name: "Kate"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	other.Username = strPtr("jack")
	if _, err := service.Update(ctx, other); !errors.Is(err, user.ErrUsernameTaken) {
		t.Fatalf("Update() error = %v, want %v", err, user.ErrUsernameTaken)
	}
}

func TestService_Update_RoleAndPoints_AreNotWritten(t *testing.T) {
	pool := testutil.NewPool(t)
	service := user.NewService(postgres.NewUserRepository(pool))
	ctx := context.Background()

	created, err := service.Create(ctx, user.User{Email: "liam@example.com", Password: "supersecret", Name: "Liam"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	created.Role = user.RoleAdmin
	created.TotalPoints = 1000
	created.DisplayName = strPtr("Liam L.")
	updated, err := service.Update(ctx, created)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Role != user.RoleUser || updated.TotalPoints != 0 {
		t.Fatalf("Update() role/points = %q/%d, want user/0", updated.Role, updated.TotalPoints)
	}
	if updated.DisplayName == nil || *updated.DisplayName != "Liam L." {
		t.Fatalf("Update() DisplayName = %v, want Liam L.", updated.DisplayName)
	}
}

func TestService_Authenticate_ValidCredentials_ReturnsUserWithoutPassword(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	created, err := service.Create(ctx, user.User{Email: "mia@example.com", Password: "supersecret", Name: "Mia"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := service.Authenticate(ctx, "MIA@Example.COM", "supersecret")
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if got.ID != created.ID || got.Role != user.RoleUser {
		t.Fatalf("Authenticate() = %+v, want the created user", got)
	}
	if got.Password != "" {
		t.Fatal("Authenticate() must not return the password hash")
	}
}

func TestService_Authenticate_InvalidCredentials_ReturnsErrInvalidCredentials(t *testing.T) {
	pool := testutil.NewPool(t)
	service := user.NewService(postgres.NewUserRepository(pool))
	ctx := context.Background()

	if _, err := service.Create(ctx, user.User{Email: "noah@example.com", Password: "supersecret", Name: "Noah"}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO users (email, password, name) VALUES ('migrated@example.com', '!', 'Migrated')`); err != nil {
		t.Fatalf("insert migrated user: %v", err)
	}

	tests := []struct {
		name     string
		email    string
		password string
	}{
		{"wrong password", "noah@example.com", "wrongpassword"},
		{"unknown email", "nobody@example.com", "supersecret"},
		{"migrated account", "migrated@example.com", "!"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := service.Authenticate(ctx, tt.email, tt.password); !errors.Is(err, user.ErrInvalidCredentials) {
				t.Fatalf("Authenticate() error = %v, want %v", err, user.ErrInvalidCredentials)
			}
		})
	}
}

func TestUserRepository_GetByEmail_DifferentCase_ReturnsUserWithHash(t *testing.T) {
	pool := testutil.NewPool(t)
	repo := postgres.NewUserRepository(pool)
	service := user.NewService(repo)
	ctx := context.Background()

	created, err := service.Create(ctx, user.User{Email: "Olivia@Example.com", Password: "supersecret", Name: "Olivia"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := repo.GetByEmail(ctx, "olivia@EXAMPLE.com")
	if err != nil {
		t.Fatalf("GetByEmail() error = %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("GetByEmail() ID = %q, want %q", got.ID, created.ID)
	}
	if !user.CheckPassword(got.Password, "supersecret") {
		t.Fatal("GetByEmail() must return the stored password hash")
	}

	if _, err := repo.GetByEmail(ctx, "nobody@example.com"); !errors.Is(err, user.ErrNotFound) {
		t.Fatalf("GetByEmail(unknown) error = %v, want %v", err, user.ErrNotFound)
	}
}

func storedPassword(t *testing.T, pool *pgxpool.Pool, id string) string {
	t.Helper()

	var password string
	if err := pool.QueryRow(context.Background(), "SELECT password FROM users WHERE id = $1", id).Scan(&password); err != nil {
		t.Fatalf("read stored password: %v", err)
	}
	return password
}

func TestService_Create_EmailDifferingByCase_ReturnsErrEmailTaken(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	created, err := service.Create(ctx, user.User{Email: " Grace@Example.com ", Password: "supersecret", Name: "Grace"})
	if err != nil {
		t.Fatalf("first Create() error = %v", err)
	}
	if created.Email != "grace@example.com" {
		t.Fatalf("stored email = %q, want %q", created.Email, "grace@example.com")
	}

	if _, err := service.Create(ctx, user.User{Email: "GRACE@example.com", Password: "supersecret", Name: "Grace 2"}); !errors.Is(err, user.ErrEmailTaken) {
		t.Fatalf("second Create() error = %v, want %v", err, user.ErrEmailTaken)
	}
}
