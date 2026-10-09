package poi

import "context"

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Get(ctx context.Context, id string) (POI, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]POI, error) {
	return s.repo.List(ctx)
}

func (s *Service) Create(ctx context.Context, e POI) (POI, error) {
	return s.repo.Create(ctx, e)
}

func (s *Service) Update(ctx context.Context, e POI) (POI, error) {
	return s.repo.Update(ctx, e)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
