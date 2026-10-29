package user

import "context"

// Service implements the user business logic on top of a Repository.
// It hashes passwords before persisting them and never returns password
// hashes to its callers.
type Service struct {
	repo Repository
}

// NewService returns a Service backed by repo.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Get returns the user identified by id, without its password hash.
func (s *Service) Get(ctx context.Context, id string) (User, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return User{}, err
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

// Delete removes the user identified by id.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// withoutPassword returns a copy of u with its password cleared.
func withoutPassword(u User) User {
	u.Password = ""
	return u
}
