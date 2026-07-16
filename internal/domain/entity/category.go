package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Category supports one level of self-reference via ParentID
// (Electronics > Phones > Smartphones). This entity intentionally
// does not carry a `Children []Category` field — GORM associations
// like that encourage accidentally loading an entire subtree on
// every query. If a "get category with its children" endpoint is
// ever needed, that's a deliberate repository method with an
// explicit Preload, not an always-there field.
type Category struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name        string    `gorm:"type:varchar(150);not null"`
	Slug        string    `gorm:"type:varchar(170);not null"`
	Description string    `gorm:"type:text;not null;default:''"`

	ParentID *uuid.UUID `gorm:"type:uuid"`

	IsActive bool `gorm:"not null;default:true"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (Category) TableName() string {
	return "categories"
}
