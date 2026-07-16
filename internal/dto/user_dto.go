package dto

import (
	"time"

	"github.com/KhalidMohomud/ecomApi/internal/domain/entity"
	"github.com/google/uuid"
)

// UserResponse is the public shape of a user — deliberately not
// entity.User. Even though entity.User already excludes
// PasswordHash from JSON via a json:"-" tag, handlers should never
// serialize an entity directly: DTOs are the one place that decides
// what a client is allowed to see, so that decision doesn't depend
// on someone remembering a struct tag three files away.
type UserResponse struct {
	ID        uuid.UUID   `json:"id"`
	Email     string      `json:"email"`
	FirstName string      `json:"first_name"`
	LastName  string      `json:"last_name"`
	Role      entity.Role `json:"role"`
	IsActive  bool        `json:"is_active"`
	CreatedAt time.Time   `json:"created_at"`
}

// NewUserResponse maps an entity to its public representation. This
// explicit field-by-field mapping is also what prevents mass
// assignment in the other direction: request DTOs (below) are mapped
// onto entity.User the same explicit way in the service layer, so a
// client can never set a field (like Role or IsActive) just by
// including an unexpected key in a JSON body.
// UpdateProfileRequest is the payload for PUT /users/me.
//
// Deliberately narrow: only the fields a user should be able to
// change about themselves. There is no Role or IsActive or Email
// field here — that absence is what stops a client from ever mass-
// assigning their way to admin, no matter what extra JSON keys they
// send, since the service layer only ever reads the fields this
// struct actually declares.
type UpdateProfileRequest struct {
	FirstName string  `json:"first_name" binding:"required,min=2,max=100"`
	LastName  string  `json:"last_name" binding:"required,min=2,max=100"`
	Phone     *string `json:"phone" binding:"omitempty,min=7,max=20"`
}

// ChangePasswordRequest is the payload for PUT /users/me/password.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

func NewUserResponse(u *entity.User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Role:      u.Role,
		IsActive:  u.IsActive,
		CreatedAt: u.CreatedAt,
	}
}
