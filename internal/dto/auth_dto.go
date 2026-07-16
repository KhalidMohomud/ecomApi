package dto

// The `binding` struct tags below are read by gin's
// c.ShouldBindJSON, which delegates to go-playground/validator
// under the hood — the same library listed in the tech stack. This
// is the normal way to use it with Gin: validation rules live right
// next to the field they constrain, instead of a separate hand-written
// validation function per request type.

// RegisterRequest is the payload for POST /auth/register.
type RegisterRequest struct {
	Email     string `json:"email" binding:"required,email" example:"jane@example.com"`
	Password  string `json:"password" binding:"required,min=8" example:"correct-horse-battery-staple"`
	FirstName string `json:"first_name" binding:"required,min=2,max=100" example:"Jane"`
	LastName  string `json:"last_name" binding:"required,min=2,max=100" example:"Doe"`
}

// LoginRequest is the payload for POST /auth/login.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email" example:"jane@example.com"`
	Password string `json:"password" binding:"required" example:"correct-horse-battery-staple"`
}

// RefreshTokenRequest is the payload for POST /auth/refresh.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// LogoutRequest is the payload for POST /auth/logout.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// AuthResponse is returned by register, login, and refresh — every
// endpoint that hands the client a new token pair.
type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	// ExpiresIn is seconds until the access token expires, so the
	// client knows when to proactively call /auth/refresh instead of
	// waiting for a 401.
	ExpiresIn int64        `json:"expires_in"`
	User      UserResponse `json:"user"`
}
