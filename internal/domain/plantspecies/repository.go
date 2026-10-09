package plantspecies

import "context"

type Repository interface {
	GetByID(ctx context.Context, id string) (PlantSpecies, error)
	List(ctx context.Context) ([]PlantSpecies, error)
	Create(ctx context.Context, e PlantSpecies) (PlantSpecies, error)
	Update(ctx context.Context, e PlantSpecies) (PlantSpecies, error)
	Delete(ctx context.Context, id string) error
}
