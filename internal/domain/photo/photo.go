package photo

import "time"

type Photo struct {
	ID         string
	UserID     string
	StorageKey string
	URL        string
	Location   []byte // WKT, nullable
	Visibility string
	NSFWStatus string
	TakenAt    *time.Time
	CreatedAt  time.Time
}
