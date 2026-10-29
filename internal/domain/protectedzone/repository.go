package protectedzone

import "context"

type Repository interface {
	GetByID(ctx context.Context, id string) (ProtectedZone, error)
	List(ctx context.Context) ([]ProtectedZone, error)
	Create(ctx context.Context, e ProtectedZone) (ProtectedZone, error)
	Update(ctx context.Context, e ProtectedZone) (ProtectedZone, error)
	Delete(ctx context.Context, id string) error
}
