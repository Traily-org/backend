package observation

import (
	"context"
	"time"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Get(ctx context.Context, id string) (Observation, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]Observation, error) {
	return s.repo.List(ctx)
}

// Create ignore le statut de validation fourni : toute nouvelle observation part en attente.
func (s *Service) Create(ctx context.Context, o Observation) (Observation, error) {
	if err := validate(o); err != nil {
		return Observation{}, err
	}
	o.ValidationStatus = "pending"
	if o.Visibility == "" {
		o.Visibility = "public"
	}
	if o.ObservedAt.IsZero() {
		o.ObservedAt = time.Now()
	}
	return s.repo.Create(ctx, o)
}

func (s *Service) Update(ctx context.Context, o Observation) (Observation, error) {
	if err := validate(o); err != nil {
		return Observation{}, err
	}
	return s.repo.Update(ctx, o)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func validate(o Observation) error {
	if len(o.Location) == 0 {
		return ErrMissingLocation
	}
	if o.ConfidenceScore != nil && (*o.ConfidenceScore < 0 || *o.ConfidenceScore > 1) {
		return ErrInvalidConfidence
	}
	return nil
}
