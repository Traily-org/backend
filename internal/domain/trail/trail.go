// Package trail implements the hiking trail domain: a recorded or curated
// route with its metadata, owned by a user and stored as a PostGIS line.
package trail

import "time"

// Difficulty rates how demanding a trail is.
type Difficulty string

// Supported trail difficulties. They mirror the trail_difficulty database enum.
const (
	DifficultyEasy     Difficulty = "easy"
	DifficultyModerate Difficulty = "moderate"
	DifficultyHard     Difficulty = "hard"
	DifficultyExpert   Difficulty = "expert"
)

// Valid reports whether d is a supported difficulty.
func (d Difficulty) Valid() bool {
	switch d {
	case DifficultyEasy, DifficultyModerate, DifficultyHard, DifficultyExpert:
		return true
	default:
		return false
	}
}

// Coordinate is a WGS 84 position in decimal degrees.
type Coordinate struct {
	Longitude float64
	Latitude  float64
}

// Trail is a hiking trail saved by a user.
type Trail struct {
	ID     string
	UserID *string
	Name   string
	// Description is optional free text.
	Description *string
	Difficulty  Difficulty
	// DistanceMeters, ElevationGainMeters and ElevationLossMeters are never negative.
	DistanceMeters      float64
	ElevationGainMeters float64
	ElevationLossMeters float64
	// EstimatedDurationSeconds is optional.
	EstimatedDurationSeconds *int
	// Path is the ordered line of the trail, at least MinPathPoints long.
	Path        []Coordinate
	IsOfficial  bool
	IsPublished bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
