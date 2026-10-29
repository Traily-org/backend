package badge

import "time"

type Badge struct {
	ID          string
	Code        string
	Name        string
	Description string
	IconURL     *string
	Criteria    []byte
	CreatedAt   time.Time
}
