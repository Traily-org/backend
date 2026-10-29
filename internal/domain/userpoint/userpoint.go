package userpoint

import "time"

type UserPoint struct {
	ID          string
	UserID      string
	Points      int
	Reason      string
	ReferenceID *string
	CreatedAt   time.Time
}
