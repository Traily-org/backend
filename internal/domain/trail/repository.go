package trail

import "context"

// Repository is the persistence port of the trail domain.
type Repository interface {
	GetByID(ctx context.Context, id string) (Trail, error)
	// ListByUserID returns the trails owned by userID, most recent first.
	ListByUserID(ctx context.Context, userID string) ([]Trail, error)
	Create(ctx context.Context, t Trail) (Trail, error)
}
