package entity

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken is a server-side record of one issued refresh token.
//
// Unlike the access token (a self-contained JWT that is never
// stored anywhere), every refresh token has a row here. That's what
// makes it revocable: logging out, changing your password, or an
// admin blocking your account can all delete the ability to use a
// refresh token by flipping RevokedAt, something that is
// impossible to do to a stateless JWT without maintaining a
// separate blocklist anyway. See internal/utils/token.go for the
// full reasoning.
type RefreshToken struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID uuid.UUID `gorm:"type:uuid;not null"`

	// SHA-256 hex digest of the raw token; the raw value is never stored.
	TokenHash string `gorm:"type:varchar(64);not null"`

	ExpiresAt time.Time `gorm:"not null"`
	RevokedAt *time.Time
	CreatedAt time.Time
}

func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

// IsValid reports whether this token can still be exchanged for a
// new session: it hasn't been revoked and hasn't expired.
func (rt *RefreshToken) IsValid() bool {
	return rt.RevokedAt == nil && time.Now().Before(rt.ExpiresAt)
}
