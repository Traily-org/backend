package dto

import (
	"time"

	"github.com/traily-org/server/internal/domain/track"
	"github.com/traily-org/server/internal/domain/user"
)

type UserResponse struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	Name        string    `json:"name"`
	Username    *string   `json:"username,omitempty"`
	DisplayName *string   `json:"display_name,omitempty"`
	Role        string    `json:"role"`
	AvatarURL   *string   `json:"avatar_url,omitempty"`
	TotalPoints int       `json:"total_points"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func NewUserResponse(u user.User) UserResponse {
	return UserResponse{
		ID:          u.ID,
		Email:       u.Email,
		Name:        u.Name,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		Role:        u.Role,
		AvatarURL:   u.AvatarURL,
		TotalPoints: u.TotalPoints,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

type CreateUserRequest struct {
	Email       string  `json:"email" binding:"required,email"`
	Password    string  `json:"password" binding:"required,min=8"`
	Name        string  `json:"name" binding:"required"`
	Username    *string `json:"username" binding:"omitempty,min=3,max=50"`
	DisplayName *string `json:"display_name" binding:"omitempty,max=100"`
	AvatarURL   *string `json:"avatar_url"`
}

type UpdateUserRequest struct {
	Email       string  `json:"email" binding:"required,email"`
	Password    string  `json:"password" binding:"required,min=8"`
	Name        string  `json:"name" binding:"required"`
	Username    *string `json:"username" binding:"omitempty,min=3,max=50"`
	DisplayName *string `json:"display_name" binding:"omitempty,max=100"`
	AvatarURL   *string `json:"avatar_url"`
}

// Track DTOs
type TrackResponse struct {
	ID             string     `json:"id"`
	UserID         string     `json:"user_id"`
	TrailID        *string    `json:"trail_id,omitempty"`
	Route          string     `json:"route,omitempty"`
	DistanceM      *float64   `json:"distance_m,omitempty"`
	ElevationGainM *float64   `json:"elevation_gain_m,omitempty"`
	ElevationLossM *float64   `json:"elevation_loss_m,omitempty"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	EndedAt        *time.Time `json:"ended_at,omitempty"`
	IsPrivate      bool       `json:"is_private"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func NewTrackResponse(t track.Track) TrackResponse {
	return TrackResponse{
		ID:             t.ID,
		UserID:         t.UserID,
		TrailID:        t.TrailID,
		Route:          string(t.Route),
		DistanceM:      t.DistanceM,
		ElevationGainM: t.ElevationGainM,
		ElevationLossM: t.ElevationLossM,
		StartedAt:      t.StartedAt,
		EndedAt:        t.EndedAt,
		IsPrivate:      t.IsPrivate,
		CreatedAt:      t.CreatedAt,
		UpdatedAt:      t.UpdatedAt,
	}
}
