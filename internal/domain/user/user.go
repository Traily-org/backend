package user

import "time"

type User struct {
	ID          string
	Email       string
	Password    string
	Name        string
	Username    *string
	DisplayName *string
	Role        string
	AvatarURL   *string
	TotalPoints int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
