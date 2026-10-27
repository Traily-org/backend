package species

import "context"

type Repository interface {
	GetByID(ctx context.Context, id string) (Species, error)
	List(ctx context.Context) ([]Species, error)
	Create(ctx context.Context, s Species) (Species, error)
	Update(ctx context.Context, s Species) (Species, error)
	Delete(ctx context.Context, id string) error
}
