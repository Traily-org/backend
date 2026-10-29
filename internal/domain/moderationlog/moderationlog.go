package moderationlog

import "time"

type ModerationLog struct {
	ID          string
	ModeratorID string
	Action      string
	TargetType  string
	TargetID    string
	Notes       *string
	CreatedAt   time.Time
}
