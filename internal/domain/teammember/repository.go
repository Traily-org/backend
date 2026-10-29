package teammember

import "context"

type Repository interface {
	GetByID(ctx context.Context, id string) (TeamMember, error)
	List(ctx context.Context) ([]TeamMember, error)
	Create(ctx context.Context, e TeamMember) (TeamMember, error)
	Update(ctx context.Context, e TeamMember) (TeamMember, error)
	Delete(ctx context.Context, id string) error
}
