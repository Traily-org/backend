package poi

import "time"

type POI struct {
	ID          string
	UserID      *string
	Name        string
	Description *string
	Type        string
	Location    []byte
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
