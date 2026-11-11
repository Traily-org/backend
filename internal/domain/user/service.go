package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// dummyPasswordHash is a precomputed bcrypt hash of a random password, using
// BcryptCost. Authenticate compares against it when the account is unknown or
// has no usable hash, so that the response time does not reveal whether an
// account exists.
const dummyPasswordHash = "$2a$12$aI4zZuAz2RHBgz14G83ry.pbDScnynBSz0kW58.txhLx2qFU3KsH."

// Service implements the user business logic on top of a Repository.
// It hashes passwords before persisting them and never returns password
// hashes to its callers.
type Service struct {
	repo     Repository
	sessions SessionRevoker
}

// SessionRevoker ends every open session of a user. It is implemented by the
// auth service, which owns the refresh tokens.
type SessionRevoker interface {
	RevokeAllSessions(ctx context.Context, userID string) error
}

// NewService returns a Service backed by repo.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// SetSessionRevoker registers the revoker called by UpdateProfile after a
// password change. It is set after construction because the auth service
// itself depends on the user service.
func (s *Service) SetSessionRevoker(r SessionRevoker) {
	s.sessions = r
}

// Get returns the user identified by id, without its password hash.
func (s *Service) Get(ctx context.Context, id string) (User, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return User{}, err
	}

	return withoutPassword(u), nil
}

// Authenticate returns the user whose email matches email (ignoring case) and
// whose password matches password, without its password hash. It returns
// ErrInvalidCredentials when the email is unknown, the password is wrong or
// the account has no usable password (e.g. accounts migrated with '!').
func (s *Service) Authenticate(ctx context.Context, email, password string) (User, error) {
	u, err := s.repo.GetByEmail(ctx, NormalizeEmail(email))
	if errors.Is(err, ErrNotFound) {
		CheckPassword(dummyPasswordHash, password)
		return User{}, ErrInvalidCredentials
	}
	if err != nil {
		return User{}, err
	}

	if !isBcryptHash(u.Password) {
		CheckPassword(dummyPasswordHash, password)
		return User{}, ErrInvalidCredentials
	}
	if !CheckPassword(u.Password, password) {
		return User{}, ErrInvalidCredentials
	}

	return withoutPassword(u), nil
}

// Create hashes the password of u and persists the user.
// It returns ErrPasswordRequired if u.Password is empty. The returned user
// does not contain the password hash.
func (s *Service) Create(ctx context.Context, u User) (User, error) {
	if u.Password == "" {
		return User{}, ErrPasswordRequired
	}
	u.Email = NormalizeEmail(u.Email)

	hash, err := HashPassword(u.Password)
	if err != nil {
		return User{}, err
	}
	u.Password = hash

	created, err := s.repo.Create(ctx, u)
	if err != nil {
		return User{}, err
	}

	return withoutPassword(created), nil
}

// Update persists the changes of u. If u.Password is not empty it is hashed
// and replaces the stored password; otherwise the stored password is kept.
// The returned user does not contain the password hash.
func (s *Service) Update(ctx context.Context, u User) (User, error) {
	u.Email = NormalizeEmail(u.Email)
	if u.Password != "" {
		hash, err := HashPassword(u.Password)
		if err != nil {
			return User{}, err
		}
		u.Password = hash
	}

	updated, err := s.repo.Update(ctx, u)
	if err != nil {
		return User{}, err
	}

	return withoutPassword(updated), nil
}

// UpdateProfile is Update for a user editing their own account. Changing the
// email or the password requires currentPassword, so that a stolen access
// token is not enough to take over the account. After a password change,
// every session of the user is revoked.
func (s *Service) UpdateProfile(ctx context.Context, u User, currentPassword string) (User, error) {
	existing, err := s.repo.GetByID(ctx, u.ID)
	if err != nil {
		return User{}, err
	}

	passwordChanged := u.Password != ""
	if passwordChanged || NormalizeEmail(u.Email) != existing.Email {
		if currentPassword == "" {
			return User{}, ErrCurrentPasswordRequired
		}
		if !isBcryptHash(existing.Password) || !CheckPassword(existing.Password, currentPassword) {
			return User{}, ErrWrongCurrentPassword
		}
	}

	updated, err := s.Update(ctx, u)
	if err != nil {
		return User{}, err
	}

	if passwordChanged && s.sessions != nil {
		if err := s.sessions.RevokeAllSessions(ctx, u.ID); err != nil {
			return User{}, fmt.Errorf("revoke sessions after password change: %w", err)
		}
	}

	return updated, nil
}

// Delete removes the user identified by id.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// isBcryptHash reports whether hash is a well-formed bcrypt hash.
func isBcryptHash(hash string) bool {
	_, err := bcrypt.Cost([]byte(hash))
	return err == nil
}

// withoutPassword returns a copy of u with its password cleared.
func withoutPassword(u User) User {
	u.Password = ""
	return u
}

// NormalizeEmail returns email trimmed and lowercased, the form in which
// emails are stored and looked up.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
