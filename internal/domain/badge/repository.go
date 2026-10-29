package badge

import "context"

type Repository interface {
	GetByID(ctx context.Context, id string) (Badge, error)
	List(ctx context.Context) ([]Badge, error)
	Create(ctx context.Context, e Badge) (Badge, error)
	Update(ctx context.Context, e Badge) (Badge, error)
	Delete(ctx context.Context, id string) error
}
