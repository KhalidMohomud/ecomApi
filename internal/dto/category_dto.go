package dto

import (
	"time"

	"github.com/KhalidMohomud/ecomApi/internal/domain/entity"
	"github.com/google/uuid"
)

// CreateCategoryRequest deliberately makes Slug optional: the
// service auto-generates one from Name via utils.Slugify when it's
// left blank, and normalizes it (via the same function) even when
// the client does supply one — see CategoryService.Create.
type CreateCategoryRequest struct {
	Name        string     `json:"name" binding:"required,min=2,max=150"`
	Slug        string     `json:"slug" binding:"omitempty,max=170"`
	Description string     `json:"description" binding:"omitempty,max=2000"`
	ParentID    *uuid.UUID `json:"parent_id"`
}

// UpdateCategoryRequest is a full replace, not a partial patch — the
// same convention UpdateProfileRequest established: every field is
// required except the genuinely optional ones (Slug, ParentID).
type UpdateCategoryRequest struct {
	Name        string     `json:"name" binding:"required,min=2,max=150"`
	Slug        string     `json:"slug" binding:"omitempty,max=170"`
	Description string     `json:"description" binding:"omitempty,max=2000"`
	ParentID    *uuid.UUID `json:"parent_id"`
	IsActive    bool       `json:"is_active"`
}

type CategoryResponse struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	Description string     `json:"description"`
	ParentID    *uuid.UUID `json:"parent_id,omitempty"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
}

func NewCategoryResponse(c *entity.Category) CategoryResponse {
	return CategoryResponse{
		ID:          c.ID,
		Name:        c.Name,
		Slug:        c.Slug,
		Description: c.Description,
		ParentID:    c.ParentID,
		IsActive:    c.IsActive,
		CreatedAt:   c.CreatedAt,
	}
}
