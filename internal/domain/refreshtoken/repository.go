package refreshtoken

import "context"

type Repository interface {
	GetByID(ctx context.Context, id string) (RefreshToken, error)
	List(ctx context.Context) ([]RefreshToken, error)
	Create(ctx context.Context, e RefreshToken) (RefreshToken, error)
	Update(ctx context.Context, e RefreshToken) (RefreshToken, error)
	Delete(ctx context.Context, id string) error
}
