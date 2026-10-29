package moderationlog

import "context"

type Repository interface {
	GetByID(ctx context.Context, id string) (ModerationLog, error)
	List(ctx context.Context) ([]ModerationLog, error)
	Create(ctx context.Context, e ModerationLog) (ModerationLog, error)
	Update(ctx context.Context, e ModerationLog) (ModerationLog, error)
	Delete(ctx context.Context, id string) error
}
