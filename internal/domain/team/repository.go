package team

import "context"

type Repository interface {
	GetByID(ctx context.Context, id string) (Team, error)
	List(ctx context.Context) ([]Team, error)
	Create(ctx context.Context, e Team) (Team, error)
	Update(ctx context.Context, e Team) (Team, error)
	Delete(ctx context.Context, id string) error
}
