package userbadge

import "context"

type Repository interface {
	GetByID(ctx context.Context, id string) (UserBadge, error)
	List(ctx context.Context) ([]UserBadge, error)
	Create(ctx context.Context, e UserBadge) (UserBadge, error)
	Update(ctx context.Context, e UserBadge) (UserBadge, error)
	Delete(ctx context.Context, id string) error
}
