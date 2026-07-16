package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Brand struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name        string    `gorm:"type:varchar(150);not null"`
	Slug        string    `gorm:"type:varchar(170);not null"`
	Description string    `gorm:"type:text;not null;default:''"`
	LogoURL     *string   `gorm:"type:varchar(500)"`

	IsActive bool `gorm:"not null;default:true"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (Brand) TableName() string {
	return "brands"
}
