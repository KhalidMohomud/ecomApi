package dto

import (
	"time"

	"github.com/KhalidMohomud/ecomApi/internal/domain/entity"
	"github.com/google/uuid"
)

type CreateBrandRequest struct {
	Name        string  `json:"name" binding:"required,min=2,max=150"`
	Slug        string  `json:"slug" binding:"omitempty,max=170"`
	Description string  `json:"description" binding:"omitempty,max=2000"`
	LogoURL     *string `json:"logo_url" binding:"omitempty,max=500"`
}

type UpdateBrandRequest struct {
	Name        string  `json:"name" binding:"required,min=2,max=150"`
	Slug        string  `json:"slug" binding:"omitempty,max=170"`
	Description string  `json:"description" binding:"omitempty,max=2000"`
	LogoURL     *string `json:"logo_url" binding:"omitempty,max=500"`
	IsActive    bool    `json:"is_active"`
}

type BrandResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	LogoURL     *string   `json:"logo_url,omitempty"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

func NewBrandResponse(b *entity.Brand) BrandResponse {
	return BrandResponse{
		ID:          b.ID,
		Name:        b.Name,
		Slug:        b.Slug,
		Description: b.Description,
		LogoURL:     b.LogoURL,
		IsActive:    b.IsActive,
		CreatedAt:   b.CreatedAt,
	}
}
