package waypoint

import "context"

type Repository interface {
	GetByID(ctx context.Context, id string) (Waypoint, error)
	List(ctx context.Context) ([]Waypoint, error)
	Create(ctx context.Context, e Waypoint) (Waypoint, error)
	Update(ctx context.Context, e Waypoint) (Waypoint, error)
	Delete(ctx context.Context, id string) error
}
