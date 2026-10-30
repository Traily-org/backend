package http

import (
	"time"

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
