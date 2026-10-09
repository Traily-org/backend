package herbierentry
import "time"
type HerbierEntry struct {
	ID            string
	HerbierID     string
	ActivityID    *string
	POIID         *string
	PlantSpeciesID string
	PhotoURL      *string
	Notes         *string
	Location      []byte
	ObservedAt    *time.Time
	CreatedAt     time.Time
}
