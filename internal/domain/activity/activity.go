package activity
import "time"
type Activity struct {
	ID        string
	UserID    string
	TrackID   string
	TraceID   *string
	Name      *string
	StartedAt *time.Time
	EndedAt   *time.Time
	CreatedAt time.Time
}
