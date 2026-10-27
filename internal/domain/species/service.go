package species

import "context"

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Get(ctx context.Context, id string) (Species, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]Species, error) {
	return s.repo.List(ctx)
}

func (s *Service) Create(ctx context.Context, sp Species) (Species, error) {
	if err := validate(sp); err != nil {
		return Species{}, err
	}
	return s.repo.Create(ctx, withDefaults(sp))
}

func (s *Service) Update(ctx context.Context, sp Species) (Species, error) {
	if err := validate(sp); err != nil {
		return Species{}, err
	}
	return s.repo.Update(ctx, withDefaults(sp))
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func withDefaults(sp Species) Species {
	if sp.ProtectionStatus == "" {
		sp.ProtectionStatus = "none"
	}
	if sp.Rarity == "" {
		sp.Rarity = "common"
	}
	return sp
}

func validate(sp Species) error {
	if sp.Category != CategoryPlant && sp.Category != CategoryFungus {
		return ErrInvalidCategory
	}
	for _, m := range sp.ObservationMonths {
		if m < 1 || m > 12 {
			return ErrInvalidMonth
		}
	}
	return nil
}
