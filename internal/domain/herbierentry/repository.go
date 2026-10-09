package herbierentry

import "context"

type Repository interface {
	GetByID(ctx context.Context, id string) (HerbierEntry, error)
	List(ctx context.Context) ([]HerbierEntry, error)
	Create(ctx context.Context, e HerbierEntry) (HerbierEntry, error)
	Update(ctx context.Context, e HerbierEntry) (HerbierEntry, error)
	Delete(ctx context.Context, id string) error
}
