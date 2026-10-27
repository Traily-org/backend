package dto

import (
	"time"

	"github.com/traily-org/server/internal/domain/appuser"
	"github.com/traily-org/server/internal/domain/observation"
	"github.com/traily-org/server/internal/domain/photo"
	"github.com/traily-org/server/internal/domain/species"
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

// Species DTOs
type SpeciesResponse struct {
	ID                string    `json:"id"`
	Category          string    `json:"category"`
	CommonName        string    `json:"common_name"`
	ScientificName    *string   `json:"scientific_name,omitempty"`
	Family            *string   `json:"family,omitempty"`
	Description       *string   `json:"description,omitempty"`
	ProtectionStatus  string    `json:"protection_status"`
	Rarity            string    `json:"rarity"`
	EthicalAdvice     *string   `json:"ethical_advice,omitempty"`
	ObservationMonths []int32   `json:"observation_months"`
	ThumbnailURL      *string   `json:"thumbnail_url,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func NewSpeciesResponse(s species.Species) SpeciesResponse {
	return SpeciesResponse{
		ID:                s.ID,
		Category:          string(s.Category),
		CommonName:        s.CommonName,
		ScientificName:    s.ScientificName,
		Family:            s.Family,
		Description:       s.Description,
		ProtectionStatus:  s.ProtectionStatus,
		Rarity:            s.Rarity,
		EthicalAdvice:     s.EthicalAdvice,
		ObservationMonths: s.ObservationMonths,
		ThumbnailURL:      s.ThumbnailURL,
		CreatedAt:         s.CreatedAt,
		UpdatedAt:         s.UpdatedAt,
	}
}

// Photo DTOs
type PhotoResponse struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	StorageKey string     `json:"storage_key"`
	URL        string     `json:"url"`
	Location   string     `json:"location,omitempty"`
	Visibility string     `json:"visibility"`
	NSFWStatus string     `json:"nsfw_status"`
	TakenAt    *time.Time `json:"taken_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

func NewPhotoResponse(p photo.Photo) PhotoResponse {
	return PhotoResponse{
		ID:         p.ID,
		UserID:     p.UserID,
		StorageKey: p.StorageKey,
		URL:        p.URL,
		Location:   string(p.Location),
		Visibility: p.Visibility,
		NSFWStatus: p.NSFWStatus,
		TakenAt:    p.TakenAt,
		CreatedAt:  p.CreatedAt,
	}
}

// Observation DTOs
type ObservationResponse struct {
	ID               string    `json:"id"`
	UserID           string    `json:"user_id"`
	SpeciesID        *string   `json:"species_id,omitempty"`
	PhotoID          *string   `json:"photo_id,omitempty"`
	TraceID          *string   `json:"trace_id,omitempty"`
	ActivityID       *string   `json:"activity_id,omitempty"`
	Location         string    `json:"location"`
	Altitude         *float64  `json:"altitude,omitempty"`
	ConfidenceScore  *float64  `json:"confidence_score,omitempty"`
	ValidationStatus string    `json:"validation_status"`
	Visibility       string    `json:"visibility"`
	ObservedAt       time.Time `json:"observed_at"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func NewObservationResponse(o observation.Observation) ObservationResponse {
	return ObservationResponse{
		ID:               o.ID,
		UserID:           o.UserID,
		SpeciesID:        o.SpeciesID,
		PhotoID:          o.PhotoID,
		TraceID:          o.TraceID,
		ActivityID:       o.ActivityID,
		Location:         string(o.Location),
		Altitude:         o.Altitude,
		ConfidenceScore:  o.ConfidenceScore,
		ValidationStatus: o.ValidationStatus,
		Visibility:       o.Visibility,
		ObservedAt:       o.ObservedAt,
		CreatedAt:        o.CreatedAt,
		UpdatedAt:        o.UpdatedAt,
	}
}
