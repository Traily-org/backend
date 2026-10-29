package observation

import "context"

type Repository interface {
	GetByID(ctx context.Context, id string) (Observation, error)
	List(ctx context.Context) ([]Observation, error)
	Create(ctx context.Context, e Observation) (Observation, error)
	Update(ctx context.Context, e Observation) (Observation, error)
	Delete(ctx context.Context, id string) error
}
