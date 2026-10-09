package appuser

import "context"

type Repository interface {
	GetByID(ctx context.Context, id string) (AppUser, error)
	GetByUsername(ctx context.Context, username string) (AppUser, error)
	List(ctx context.Context) ([]AppUser, error)
	Create(ctx context.Context, u AppUser) (AppUser, error)
	Update(ctx context.Context, u AppUser) (AppUser, error)
	Delete(ctx context.Context, id string) error
}
