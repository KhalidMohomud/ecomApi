package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Product stores price as PriceCents (an integer count of the
// currency's smallest unit) rather than a float, for the reason
// explained in the migration: floats cannot represent most decimal
// fractions exactly, and that error compounds badly across a cart of
// items or a large catalog. Every layer above this — DTOs, requests,
// responses — deals in cents too; the only place a fractional dollar
// amount should ever appear is client-side display formatting.
type Product struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name        string    `gorm:"type:varchar(200);not null"`
	Slug        string    `gorm:"type:varchar(220);not null"`
	SKU         string    `gorm:"type:varchar(64);not null"`
	Description string    `gorm:"type:text;not null;default:''"`
	PriceCents  int64     `gorm:"not null"`

	CategoryID uuid.UUID  `gorm:"type:uuid;not null"`
	BrandID    *uuid.UUID `gorm:"type:uuid"`

	IsActive bool `gorm:"not null;default:true"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (Product) TableName() string {
	return "products"
}
