package plantspecies

import "time"

type PlantSpecies struct {
	ID             string
	CommonName     string
	ScientificName *string
	Description    *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
