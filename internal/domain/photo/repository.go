package photo

import "context"

type Repository interface {
	GetByID(ctx context.Context, id string) (Photo, error)
	List(ctx context.Context) ([]Photo, error)
	Create(ctx context.Context, e Photo) (Photo, error)
	Update(ctx context.Context, e Photo) (Photo, error)
	Delete(ctx context.Context, id string) error
}
