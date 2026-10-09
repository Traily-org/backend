package track

import "context"

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Get(ctx context.Context, id string) (Track, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ListByUser(ctx context.Context, userID string) ([]Track, error) {
	return s.repo.ListByUserID(ctx, userID)
}

func (s *Service) Create(ctx context.Context, t Track) (Track, error) {
	if t.DistanceM != nil && *t.DistanceM < 0 {
		return Track{}, ErrInvalidDistance
	}
	if t.ElevationGainM != nil && *t.ElevationGainM < 0 {
		return Track{}, ErrInvalidElevation
	}
	if t.ElevationLossM != nil && *t.ElevationLossM < 0 {
		return Track{}, ErrInvalidElevation
	}
	if t.StartedAt != nil && t.EndedAt != nil && t.EndedAt.Before(*t.StartedAt) {
		return Track{}, ErrInvalidDates
	}
	return s.repo.Create(ctx, t)
}

func (s *Service) Update(ctx context.Context, t Track) (Track, error) {
	if t.DistanceM != nil && *t.DistanceM < 0 {
		return Track{}, ErrInvalidDistance
	}
	if t.ElevationGainM != nil && *t.ElevationGainM < 0 {
		return Track{}, ErrInvalidElevation
	}
	if t.ElevationLossM != nil && *t.ElevationLossM < 0 {
		return Track{}, ErrInvalidElevation
	}
	if t.StartedAt != nil && t.EndedAt != nil && t.EndedAt.Before(*t.StartedAt) {
		return Track{}, ErrInvalidDates
	}
	return s.repo.Update(ctx, t)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
