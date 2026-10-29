package userpoint

import "context"

type Repository interface {
	GetByID(ctx context.Context, id string) (UserPoint, error)
	List(ctx context.Context) ([]UserPoint, error)
	Create(ctx context.Context, e UserPoint) (UserPoint, error)
	Update(ctx context.Context, e UserPoint) (UserPoint, error)
	Delete(ctx context.Context, id string) error
}
