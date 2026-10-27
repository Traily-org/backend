package species_test

import (
	"context"
	"errors"
	"testing"

	"github.com/traily-org/server/internal/domain/species"
)

type fakeRepo struct{ species.Repository }

func (fakeRepo) Create(_ context.Context, s species.Species) (species.Species, error) { return s, nil }

func TestService_Create_Validation(t *testing.T) {
	svc := species.NewService(fakeRepo{})
	tests := []struct {
		name string
		in   species.Species
		want error
	}{
		{"plant ok", species.Species{Category: species.CategoryPlant, CommonName: "Ortie"}, nil},
		{"fungus ok", species.Species{Category: species.CategoryFungus, CommonName: "Cèpe", ObservationMonths: []int32{9, 10}}, nil},
		{"fauna rejected", species.Species{Category: "fauna", CommonName: "Renard"}, species.ErrInvalidCategory},
		{"empty category rejected", species.Species{CommonName: "?"}, species.ErrInvalidCategory},
		{"month 13 rejected", species.Species{Category: species.CategoryPlant, ObservationMonths: []int32{13}}, species.ErrInvalidMonth},
		{"month 0 rejected", species.Species{Category: species.CategoryPlant, ObservationMonths: []int32{0}}, species.ErrInvalidMonth},
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

func TestService_Create_Defaults(t *testing.T) {
	got, err := species.NewService(fakeRepo{}).Create(context.Background(), species.Species{Category: species.CategoryPlant})
	if err != nil {
		t.Fatal(err)
	}
	if got.ProtectionStatus != "none" || got.Rarity != "common" {
		t.Fatalf("defaults = %q/%q, want none/common", got.ProtectionStatus, got.Rarity)
	}
}
