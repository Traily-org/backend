package userbadge

import "time"

type UserBadge struct {
	ID         string
	UserID     string
	BadgeID    string
	UnlockedAt time.Time
}
