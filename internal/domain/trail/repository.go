package trail

import "context"

type Repository interface {
	GetByID(ctx context.Context, id string) (Trail, error)
	List(ctx context.Context) ([]Trail, error)
	Create(ctx context.Context, e Trail) (Trail, error)
	Update(ctx context.Context, e Trail) (Trail, error)
	Delete(ctx context.Context, id string) error
}
