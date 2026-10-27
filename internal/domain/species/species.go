package species

import "time"

type Category string

const (
	CategoryPlant  Category = "plant"
	CategoryFungus Category = "fungus"
)

type Species struct {
	ID                string
	Category          Category
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
