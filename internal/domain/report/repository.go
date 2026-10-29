package report

import "context"

type Repository interface {
	GetByID(ctx context.Context, id string) (Report, error)
	List(ctx context.Context) ([]Report, error)
	Create(ctx context.Context, e Report) (Report, error)
	Update(ctx context.Context, e Report) (Report, error)
	Delete(ctx context.Context, id string) error
}
