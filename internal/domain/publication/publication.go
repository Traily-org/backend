package publication
import "time"
type Publication struct {
	ID          string
	UserID      string
	TrackID     string
	TraceID     *string
	Title       *string
	Content     *string
	PublishedAt time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
