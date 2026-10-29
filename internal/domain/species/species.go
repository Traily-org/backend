package species

import "time"

type Species struct {
	ID                string
	Category          string
	CommonName        string
	ScientificName    *string
	Family            *string
	Description       *string
	ProtectionStatus  string
	Rarity            string
	EthicalAdvice     *string
	ObservationMonths []int32
	ThumbnailURL      *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
