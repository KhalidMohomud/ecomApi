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

func (r *brandRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&entity.Brand{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete brand: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("delete brand: %w", entity.ErrNotFound)
	}
	return nil
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
