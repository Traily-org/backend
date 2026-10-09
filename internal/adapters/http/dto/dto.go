package dto

import (
	"time"

	"github.com/traily-org/server/internal/domain/appuser"
	"github.com/traily-org/server/internal/domain/track"
	"github.com/traily-org/server/internal/domain/user"
)

type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewUserResponse(u user.User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

type CreateUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name" binding:"required"`
}

type UpdateUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name" binding:"required"`
}

// AppUser DTOs
type AppUserResponse struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	DisplayName *string   `json:"display_name,omitempty"`
	AvatarURL   *string   `json:"avatar_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func NewAppUserResponse(u appuser.AppUser) AppUserResponse {
	return AppUserResponse{
		ID:          u.ID,
		Username:    u.Username,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		AvatarURL:   u.AvatarURL,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

type CreateAppUserRequest struct {
	Username    string  `json:"username" binding:"required,min=3,max=50"`
	Email       string  `json:"email" binding:"required,email"`
	DisplayName *string `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
}

type UpdateAppUserRequest struct {
	Username    string  `json:"username" binding:"required,min=3,max=50"`
	Email       string  `json:"email" binding:"required,email"`
	DisplayName *string `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
}

// Track DTOs
type TrackResponse struct {
	ID             string     `json:"id"`
	UserID         string     `json:"user_id"`
	Route          string     `json:"route,omitempty"`
	DistanceM      *float64   `json:"distance_m,omitempty"`
	ElevationGainM *float64   `json:"elevation_gain_m,omitempty"`
	ElevationLossM *float64   `json:"elevation_loss_m,omitempty"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	EndedAt        *time.Time `json:"ended_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

func NewTrackResponse(t track.Track) TrackResponse {
	return TrackResponse{
		ID:             t.ID,
		UserID:         t.UserID,
		Route:          string(t.Route),
		DistanceM:      t.DistanceM,
		ElevationGainM: t.ElevationGainM,
		ElevationLossM: t.ElevationLossM,
		StartedAt:      t.StartedAt,
		EndedAt:        t.EndedAt,
		CreatedAt:      t.CreatedAt,
	}
}

// Generic DTOs for Trace, Activity, Publication, POI, Herbier, PlantSpecies, HerbierEntry
// These are minimal implementations - expand as needed
type entityResponse struct {
	ID        string      `json:"id"`
	Data      interface{} `json:"data,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
}
