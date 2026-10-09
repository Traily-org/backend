package poi

import "context"

type Repository interface {
	GetByID(ctx context.Context, id string) (POI, error)
	List(ctx context.Context) ([]POI, error)
	Create(ctx context.Context, e POI) (POI, error)
	Update(ctx context.Context, e POI) (POI, error)
	Delete(ctx context.Context, id string) error
}
