package track

import "context"

type Repository interface {
	GetByID(ctx context.Context, id string) (Track, error)
	ListByUserID(ctx context.Context, userID string) ([]Track, error)
	Create(ctx context.Context, t Track) (Track, error)
	Update(ctx context.Context, t Track) (Track, error)
	Delete(ctx context.Context, id string) error
}
