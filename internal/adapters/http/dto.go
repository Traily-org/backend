package http

import (
	"time"

	"github.com/traily-org/server/internal/domain/trail"
	"github.com/traily-org/server/internal/domain/user"
)

// userResponse is the public representation of a user. Dates are UTC.
type userResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func newUserResponse(u user.User) userResponse {
	return userResponse{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		CreatedAt: u.CreatedAt.UTC(),
		UpdatedAt: u.UpdatedAt.UTC(),
	}
}

// createUserRequest is the payload of POST /users.
type createUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
	Name     string `json:"name" binding:"required"`
}

// updateUserRequest is the payload of PUT /users/:id. Password is optional:
// when omitted, the current password is kept. The 72-character cap rejects
// most over-long passwords early; the 72-byte bcrypt limit itself is enforced
// by the user domain (multi-byte passwords can pass this binding).
type updateUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"omitempty,min=8,max=72"`
	Name     string `json:"name" binding:"required"`
}

// geoJSONLineString is a GeoJSON LineString: coordinates are [longitude,
// latitude] pairs. A third value (altitude) is accepted on input and ignored.
type geoJSONLineString struct {
	Type        string      `json:"type" binding:"required,eq=LineString"`
	Coordinates [][]float64 `json:"coordinates" binding:"required,min=2,dive,min=2,max=3"`
}

func newGeoJSONLineString(path []trail.Coordinate) geoJSONLineString {
	coordinates := make([][]float64, len(path))
	for i, c := range path {
		coordinates[i] = []float64{c.Longitude, c.Latitude}
	}

	return geoJSONLineString{Type: "LineString", Coordinates: coordinates}
}

func (l geoJSONLineString) path() []trail.Coordinate {
	path := make([]trail.Coordinate, len(l.Coordinates))
	for i, c := range l.Coordinates {
		path[i] = trail.Coordinate{Longitude: c[0], Latitude: c[1]}
	}

	return path
}

// trailResponse is the public representation of a trail. Dates are UTC.
type trailResponse struct {
	ID                       string            `json:"id"`
	UserID                   *string           `json:"user_id"`
	Name                     string            `json:"name"`
	Description              *string           `json:"description"`
	Difficulty               string            `json:"difficulty"`
	DistanceMeters           float64           `json:"distance_meters"`
	ElevationGainMeters      float64           `json:"elevation_gain_meters"`
	ElevationLossMeters      float64           `json:"elevation_loss_meters"`
	EstimatedDurationSeconds *int              `json:"estimated_duration_seconds"`
	Path                     geoJSONLineString `json:"path"`
	IsOfficial               bool              `json:"is_official"`
	IsPublished              bool              `json:"is_published"`
	CreatedAt                time.Time         `json:"created_at"`
	UpdatedAt                time.Time         `json:"updated_at"`
}

func newTrailResponse(t trail.Trail) trailResponse {
	return trailResponse{
		ID:                       t.ID,
		UserID:                   t.UserID,
		Name:                     t.Name,
		Description:              t.Description,
		Difficulty:               string(t.Difficulty),
		DistanceMeters:           t.DistanceMeters,
		ElevationGainMeters:      t.ElevationGainMeters,
		ElevationLossMeters:      t.ElevationLossMeters,
		EstimatedDurationSeconds: t.EstimatedDurationSeconds,
		Path:                     newGeoJSONLineString(t.Path),
		IsOfficial:               t.IsOfficial,
		IsPublished:              t.IsPublished,
		CreatedAt:                t.CreatedAt.UTC(),
		UpdatedAt:                t.UpdatedAt.UTC(),
	}
}

// createTrailRequest is the payload of POST /trails. Metrics are pointers so
// that a missing value is rejected while an explicit 0 (e.g. no elevation
// loss) is accepted. is_official is deliberately not settable by clients.
type createTrailRequest struct {
	UserID                   string             `json:"user_id" binding:"required,uuid"`
	Name                     string             `json:"name" binding:"required,max=200"`
	Description              *string            `json:"description"`
	Difficulty               string             `json:"difficulty" binding:"omitempty,oneof=easy moderate hard expert"`
	DistanceMeters           *float64           `json:"distance_meters" binding:"required,gte=0"`
	ElevationGainMeters      *float64           `json:"elevation_gain_meters" binding:"required,gte=0"`
	ElevationLossMeters      *float64           `json:"elevation_loss_meters" binding:"required,gte=0"`
	EstimatedDurationSeconds *int               `json:"estimated_duration_seconds" binding:"omitempty,gte=0"`
	Path                     *geoJSONLineString `json:"path" binding:"required"`
	IsPublished              *bool              `json:"is_published"`
}

// listTrailsQuery holds the query parameters of GET /trails.
type listTrailsQuery struct {
	UserID string `form:"user_id" json:"user_id" binding:"required,uuid"`
}
