package protectedzone

import "time"

type ProtectedZone struct {
	ID          string
	Name        string
	Category    string
	Description *string
	Regulations *string
	Boundary    []byte
	CreatedAt   time.Time
}
