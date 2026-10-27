package observation_test

import (
	"context"
	"errors"
	"testing"

	"github.com/traily-org/server/internal/domain/observation"
)

type fakeRepo struct{ observation.Repository }

func (fakeRepo) Create(_ context.Context, o observation.Observation) (observation.Observation, error) {
	return o, nil
}

func ptr(f float64) *float64 { return &f }

func TestService_Create_Validation(t *testing.T) {
	svc := observation.NewService(fakeRepo{})
	loc := []byte("POINT(6.86 45.83)")
	tests := []struct {
		name string
		in   observation.Observation
		want error
	}{
		{"ok", observation.Observation{Location: loc, ConfidenceScore: ptr(0.9)}, nil},
		{"missing location", observation.Observation{}, observation.ErrMissingLocation},
		{"confidence > 1", observation.Observation{Location: loc, ConfidenceScore: ptr(1.2)}, observation.ErrInvalidConfidence},
		{"confidence < 0", observation.Observation{Location: loc, ConfidenceScore: ptr(-0.1)}, observation.ErrInvalidConfidence},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Create(context.Background(), tt.in)
			if !errors.Is(err, tt.want) {
				t.Fatalf("Create() error = %v, want %v", err, tt.want)
			}
		})
	}
}

// Une observation créée part toujours en attente, même si le client envoie "validated".
func TestService_Create_ForcesPending(t *testing.T) {
	got, err := observation.NewService(fakeRepo{}).Create(context.Background(), observation.Observation{
		Location:         []byte("POINT(0 0)"),
		ValidationStatus: "validated",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.ValidationStatus != "pending" || got.Visibility != "public" || got.ObservedAt.IsZero() {
		t.Fatalf("got status=%q visibility=%q observedAt=%v", got.ValidationStatus, got.Visibility, got.ObservedAt)
	}
}
