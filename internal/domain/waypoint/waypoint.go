package waypoint

import "time"

type Waypoint struct {
	ID          string
	TrailID     string
	Label       string
	Description *string
	OrderIndex  int
	Location    []byte
	PhotoURL    *string
	CreatedAt   time.Time
}
