package appuser

import "context"

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Get(ctx context.Context, id string) (AppUser, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) GetByUsername(ctx context.Context, username string) (AppUser, error) {
	return s.repo.GetByUsername(ctx, username)
}

func (s *Service) List(ctx context.Context) ([]AppUser, error) {
	return s.repo.List(ctx)
}

func (s *Service) Create(ctx context.Context, u AppUser) (AppUser, error) {
	return s.repo.Create(ctx, u)
}

func (s *Service) Update(ctx context.Context, u AppUser) (AppUser, error) {
	return s.repo.Update(ctx, u)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
