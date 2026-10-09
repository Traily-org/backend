package activity

import "context"

type Repository interface {
	GetByID(ctx context.Context, id string) (Activity, error)
	List(ctx context.Context) ([]Activity, error)
	Create(ctx context.Context, e Activity) (Activity, error)
	Update(ctx context.Context, e Activity) (Activity, error)
	Delete(ctx context.Context, id string) error
}
