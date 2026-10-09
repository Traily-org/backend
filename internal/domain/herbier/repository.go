package herbier

import "context"

type Repository interface {
	GetByID(ctx context.Context, id string) (Herbier, error)
	List(ctx context.Context) ([]Herbier, error)
	Create(ctx context.Context, e Herbier) (Herbier, error)
	Update(ctx context.Context, e Herbier) (Herbier, error)
	Delete(ctx context.Context, id string) error
}
