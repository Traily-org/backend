package trace

import "context"

type Repository interface {
	GetByID(ctx context.Context, id string) (Trace, error)
	List(ctx context.Context) ([]Trace, error)
	Create(ctx context.Context, e Trace) (Trace, error)
	Update(ctx context.Context, e Trace) (Trace, error)
	Delete(ctx context.Context, id string) error
}
