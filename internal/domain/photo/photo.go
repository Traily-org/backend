package photo

import "time"

type Photo struct {
	ID         string
	UserID     string
	StorageKey string
	URL        string
	Location   []byte
	Visibility string
	NSFWStatus string
	TakenAt    *time.Time
	CreatedAt  time.Time
}
