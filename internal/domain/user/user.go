// Package user holds the user domain: the User entity, its persistence
// contract and the service implementing the user business rules.
package user

import "time"

// User is a Traily account. Password holds the bcrypt hash when read from a
// Repository, the plain password when passed to Service.Create or
// Service.Update, and is always empty in values returned by the Service.
type User struct {
	ID       string
	Email    string
	Password string
	Name     string
	// Username is the unique public handle of the user, or nil when unset.
	Username *string
	// DisplayName is the name shown to other users, or nil when unset.
	DisplayName *string
	// AvatarURL is the URL of the profile picture, or nil when unset.
	AvatarURL *string
	// Role is the permission level of the user. It is read-only for the user
	// and never written by Repository.Create or Repository.Update.
	Role Role
	// TotalPoints is the gamification score, maintained by the database from
	// the user's points. It is never written by the Repository.
	TotalPoints int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
