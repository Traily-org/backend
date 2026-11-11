package user

import "context"

// Repository persists users. Implementations return ErrNotFound when no user
// matches, ErrEmailTaken or ErrUsernameTaken on uniqueness violations, and
// never write Role nor TotalPoints.
type Repository interface {
	// GetByID returns the user identified by id.
	GetByID(ctx context.Context, id string) (User, error)
	// GetByEmail returns the user whose email matches email, ignoring case.
	// The returned user contains the stored password hash.
	GetByEmail(ctx context.Context, email string) (User, error)
	// Create inserts u and returns the stored user.
	Create(ctx context.Context, u User) (User, error)
	// Update replaces the editable fields of u. An empty u.Password keeps the
	// stored password.
	Update(ctx context.Context, u User) (User, error)
	// Delete removes the user identified by id.
	Delete(ctx context.Context, id string) error
}
