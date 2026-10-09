package publication

import "context"

type Repository interface {
	GetByID(ctx context.Context, id string) (Publication, error)
	List(ctx context.Context) ([]Publication, error)
	Create(ctx context.Context, e Publication) (Publication, error)
	Update(ctx context.Context, e Publication) (Publication, error)
	Delete(ctx context.Context, id string) error
}
