package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Role represents a user's permission level. It's a distinct named
// type (defined type) instead of a bare string so the compiler
// stops you from passing an arbitrary string where a Role is
// expected, and so every valid value is enumerated in one place.
type Role string

const (
	RoleCustomer  Role = "customer"
	RoleAdmin     Role = "admin"
	RoleModerator Role = "moderator"
)

// User is the domain entity for an account.
//
// Its columns mirror the `users` table exactly, but the table
// itself is owned by the SQL files in /migrations, applied with
// golang-migrate — never by GORM's AutoMigrate. AutoMigrate can add
// columns but can't safely rename or drop them, can't manage
// partial indexes like the one on email, and gives you no rollback
// path. Plain SQL migrations give us both, at the cost of having to
// keep this struct in sync with the schema by hand.
type User struct {
	ID    uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Email string    `gorm:"type:varchar(255);not null"`

	// json:"-" is defense in depth: even if a handler ever
	// accidentally serializes a *entity.User directly instead of
	// mapping it through a DTO, the hash can never leak into a
	// response body.
	PasswordHash string `gorm:"type:varchar(255);not null" json:"-"`

	FirstName string  `gorm:"type:varchar(100);not null"`
	LastName  string  `gorm:"type:varchar(100);not null"`
	Phone     *string `gorm:"type:varchar(20)"`
	Role      Role    `gorm:"type:varchar(20);not null;default:customer"`
	IsActive  bool    `gorm:"not null;default:true"`

	// nil means "not yet verified". Set once, by the verify-email flow.
	EmailVerifiedAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time

	// gorm.DeletedAt makes this a soft-delete model: calling
	// db.Delete() on a *User sets this column instead of removing
	// the row, and GORM automatically adds "WHERE deleted_at IS
	// NULL" to every query GORM generates for this model.
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TableName pins the table name explicitly instead of relying on
// GORM's pluralization convention (User -> "users"). Being explicit
// keeps the mapping obvious and correct even if that convention
// ever changes.
func (User) TableName() string {
	return "users"
}

// IsBlocked reports whether an admin has deactivated this account.
func (u *User) IsBlocked() bool {
	return !u.IsActive
}
