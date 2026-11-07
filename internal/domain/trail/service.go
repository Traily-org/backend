package trail

import (
	"context"
	"math"
	"strings"
)

// MinPathPoints is the minimum number of points of a trail path.
const MinPathPoints = 2

// Service implements the trail business logic on top of a Repository.
type Service struct {
	repo Repository
}

// NewService returns a Service backed by repo.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Get returns the trail identified by id, or ErrNotFound.
func (s *Service) Get(ctx context.Context, id string) (Trail, error) {
	return s.repo.GetByID(ctx, id)
}

// ListByUser returns the trails saved by the given user, most recent first.
// It returns ErrUserRequired when userID is empty.
func (s *Service) ListByUser(ctx context.Context, userID string) ([]Trail, error) {
	if userID == "" {
		return nil, ErrUserRequired
	}

	return s.repo.ListByUserID(ctx, userID)
}

// Create validates t and saves it. The trail must have an owner, a name, a
// path of at least MinPathPoints valid coordinates and non-negative metrics.
// A missing difficulty defaults to DifficultyModerate. Trails created through
// the service are never official: that flag is reserved to curated content.
func (s *Service) Create(ctx context.Context, t Trail) (Trail, error) {
	t.Name = strings.TrimSpace(t.Name)
	if t.Difficulty == "" {
		t.Difficulty = DifficultyModerate
	}
	t.IsOfficial = false

	if err := validate(t); err != nil {
		return Trail{}, err
	}

	return s.repo.Create(ctx, t)
}

func validate(t Trail) error {
	switch {
	case t.UserID == nil || *t.UserID == "":
		return ErrUserRequired
	case t.Name == "":
		return ErrNameRequired
	case !t.Difficulty.Valid():
		return ErrInvalidDifficulty
	case !isFiniteNonNegative(t.DistanceMeters):
		return ErrInvalidDistance
	case !isFiniteNonNegative(t.ElevationGainMeters), !isFiniteNonNegative(t.ElevationLossMeters):
		return ErrInvalidElevation
	case t.EstimatedDurationSeconds != nil && *t.EstimatedDurationSeconds < 0:
		return ErrInvalidDuration
	case len(t.Path) < MinPathPoints:
		return ErrPathTooShort
	}

	for _, c := range t.Path {
		if !validCoordinate(c) {
			return ErrInvalidCoordinate
		}
	}

	return nil
}

func isFiniteNonNegative(v float64) bool {
	return v >= 0 && !math.IsInf(v, 0) && !math.IsNaN(v)
}

func validCoordinate(c Coordinate) bool {
	return c.Longitude >= -180 && c.Longitude <= 180 &&
		c.Latitude >= -90 && c.Latitude <= 90
}
