package appuser

import "time"

type AppUser struct {
	ID          string
	Username    string
	Email       string
	DisplayName *string // nullable
	AvatarURL   *string // nullable
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
