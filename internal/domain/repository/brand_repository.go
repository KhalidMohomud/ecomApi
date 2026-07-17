package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/KhalidMohomud/ecomApi/internal/domain/entity"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BrandRepository interface {
	Create(ctx context.Context, brand *entity.Brand) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Brand, error)
	GetBySlug(ctx context.Context, slug string) (*entity.Brand, error)
	Update(ctx context.Context, brand *entity.Brand) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, offset, limit int) ([]entity.Brand, int64, error)
}

type brandRepository struct {
	db *gorm.DB
}

func NewBrandRepository(db *gorm.DB) BrandRepository {
	return &brandRepository{db: db}
}

func (r *brandRepository) Create(ctx context.Context, brand *entity.Brand) error {
	if err := r.db.WithContext(ctx).Create(brand).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return fmt.Errorf("create brand: %w", entity.ErrConflict)
		}
		return fmt.Errorf("create brand: %w", err)
	}
	return nil
}

func (r *brandRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Brand, error) {
	var brand entity.Brand
	if err := r.db.WithContext(ctx).First(&brand, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("get brand by id: %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("get brand by id: %w", err)
	}
	return &brand, nil
}

func (r *brandRepository) GetBySlug(ctx context.Context, slug string) (*entity.Brand, error) {
	var brand entity.Brand
	if err := r.db.WithContext(ctx).First(&brand, "slug = ?", slug).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("get brand by slug: %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("get brand by slug: %w", err)
	}
	return &brand, nil
}

func (r *brandRepository) Update(ctx context.Context, brand *entity.Brand) error {
	if err := r.db.WithContext(ctx).Save(brand).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return fmt.Errorf("update brand: %w", entity.ErrConflict)
		}
		return fmt.Errorf("update brand: %w", err)
	}
	return nil
}

// Delete soft-deletes a brand and, in the same transaction, clears
// brand_id on any product that referenced it.
//
// This is the same fix as CategoryRepository.Delete's child
// promotion, applied for the same reason: the migration's `brand_id
// ... ON DELETE SET NULL` only fires on a real SQL DELETE, and this
// is a soft delete (an UPDATE), so that FK action never triggers.
// Unlike a category (which a product MUST belong to — see
// CategoryService.Delete, which refuses to delete a category with
// products at all), a brand is optional on a product, so silently
// clearing the reference here is the correct behavior rather than
// blocking the delete.
func (r *brandRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Delete(&entity.Brand{}, "id = ?", id)
		if result.Error != nil {
			return fmt.Errorf("delete brand: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("delete brand: %w", entity.ErrNotFound)
		}

		if err := tx.Model(&entity.Product{}).
			Where("brand_id = ?", id).
			Update("brand_id", nil).Error; err != nil {
			return fmt.Errorf("delete brand: clearing product references: %w", err)
		}

		return nil
	})
}

func (r *brandRepository) List(ctx context.Context, offset, limit int) ([]entity.Brand, int64, error) {
	var brands []entity.Brand
	var total int64

	if err := r.db.WithContext(ctx).Model(&entity.Brand{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count brands: %w", err)
	}

	if err := r.db.WithContext(ctx).
		Order("name ASC").
		Offset(offset).
		Limit(limit).
		Find(&brands).Error; err != nil {
		return nil, 0, fmt.Errorf("list brands: %w", err)
	}

	return brands, total, nil
}
