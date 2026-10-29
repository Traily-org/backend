package trail

import "context"

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Get(ctx context.Context, id string) (Trail, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]Trail, error) {
	return s.repo.List(ctx)
}

func (s *Service) Create(ctx context.Context, e Trail) (Trail, error) {
	return s.repo.Create(ctx, e)
}

func (s *Service) Update(ctx context.Context, e Trail) (Trail, error) {
	return s.repo.Update(ctx, e)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
