package photo

import "context"

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Get(ctx context.Context, id string) (Photo, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]Photo, error) {
	return s.repo.List(ctx)
}

// Create ignore le statut NSFW fourni : toute nouvelle photo attend la vérification IA.
func (s *Service) Create(ctx context.Context, p Photo) (Photo, error) {
	p.NSFWStatus = "pending"
	if p.Visibility == "" {
		p.Visibility = "public"
	}
	return s.repo.Create(ctx, p)
}

func (s *Service) Update(ctx context.Context, p Photo) (Photo, error) {
	return s.repo.Update(ctx, p)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
