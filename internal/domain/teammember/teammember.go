package teammember

import "time"

type TeamMember struct {
	ID       string
	TeamID   string
	UserID   string
	Role     string
	JoinedAt time.Time
}
