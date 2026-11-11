package http

import (
	"regexp"
	"time"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	"github.com/traily-org/server/internal/domain/user"
)

// usernameTag is the binding rule restricting a username to letters, digits,
// '_', '.' and '-'.
const usernameTag = "username"

// usernamePattern matches the characters allowed in a username.
var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		if err := v.RegisterValidation(usernameTag, validateUsername); err != nil {
			panic(err)
		}
	}
}

// validateUsername implements the usernameTag binding rule.
func validateUsername(fl validator.FieldLevel) bool {
	return usernamePattern.MatchString(fl.Field().String())
}

// userResponse is the representation of a user for the user themselves,
// including private fields such as the email and role. Dates are UTC.
type userResponse struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	Name        string    `json:"name"`
	Username    *string   `json:"username,omitempty"`
	DisplayName *string   `json:"display_name,omitempty"`
	AvatarURL   *string   `json:"avatar_url,omitempty"`
	Role        user.Role `json:"role"`
	TotalPoints int       `json:"total_points"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func newUserResponse(u user.User) userResponse {
	return userResponse{
		ID:          u.ID,
		Email:       u.Email,
		Name:        u.Name,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		AvatarURL:   u.AvatarURL,
		Role:        u.Role,
		TotalPoints: u.TotalPoints,
		CreatedAt:   u.CreatedAt.UTC(),
		UpdatedAt:   u.UpdatedAt.UTC(),
	}
}

// publicUserResponse is the representation of a user shown to other users.
// It never contains the email nor the role. Dates are UTC.
type publicUserResponse struct {
	ID          string    `json:"id"`
	Username    *string   `json:"username,omitempty"`
	DisplayName *string   `json:"display_name,omitempty"`
	AvatarURL   *string   `json:"avatar_url,omitempty"`
	TotalPoints int       `json:"total_points"`
	CreatedAt   time.Time `json:"created_at"`
}

func newPublicUserResponse(u user.User) publicUserResponse {
	return publicUserResponse{
		ID:          u.ID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		AvatarURL:   u.AvatarURL,
		TotalPoints: u.TotalPoints,
		CreatedAt:   u.CreatedAt.UTC(),
	}
}

// updateUserRequest is the payload of PUT /users/me. It replaces the profile
// of the authenticated user: omitted optional fields are cleared, except
// Password which keeps the current password when omitted. The 72-character
// cap rejects most over-long passwords early; the 72-byte bcrypt limit itself
// is enforced by the user domain (multi-byte passwords can pass this binding).
type updateUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"omitempty,min=8,max=72"`
	// CurrentPassword is required to change the email or the password.
	CurrentPassword string `json:"current_password" binding:"omitempty,max=72"`
	Name            string `json:"name" binding:"required"`
	Username        string `json:"username" binding:"omitempty,min=3,max=50,username"`
	DisplayName     string `json:"display_name" binding:"omitempty,max=100"`
	AvatarURL       string `json:"avatar_url" binding:"omitempty,http_url"`
}

// optional returns nil for an empty string and a pointer to s otherwise.
func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
