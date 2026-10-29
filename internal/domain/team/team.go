package team

import "time"

type Team struct {
	ID          string
	Name        string
	Description *string
	AvatarURL   *string
	CreatedBy   *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
