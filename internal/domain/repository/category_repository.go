package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/KhalidMohomud/ecomApi/internal/domain/entity"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CategoryRepository follows the exact same shape as UserRepository:
// an interface the service layer depends on, a GORM-backed struct
// implementing it, sentinel errors from the entity package on
// failure. By now that shape should look like a template rather than
// something to design from scratch each time — CategoryRepository,
// BrandRepository, and every catalog repository after it are all
// this same pattern applied to a different table.
type CategoryRepository interface {
	Create(ctx context.Context, category *entity.Category) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Category, error)
	GetBySlug(ctx context.Context, slug string) (*entity.Category, error)
	Update(ctx context.Context, category *entity.Category) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, offset, limit int) ([]entity.Category, int64, error)
}

type categoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) CategoryRepository {
	return &categoryRepository{db: db}
}

func (r *categoryRepository) Create(ctx context.Context, category *entity.Category) error {
	if err := r.db.WithContext(ctx).Create(category).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return fmt.Errorf("create category: %w", entity.ErrConflict)
		}
		return fmt.Errorf("create category: %w", err)
	}
	return nil
}

func (r *categoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Category, error) {
	var category entity.Category
	if err := r.db.WithContext(ctx).First(&category, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("get category by id: %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("get category by id: %w", err)
	}
	return &category, nil
}

func (r *categoryRepository) GetBySlug(ctx context.Context, slug string) (*entity.Category, error) {
	var category entity.Category
	if err := r.db.WithContext(ctx).First(&category, "slug = ?", slug).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("get category by slug: %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("get category by slug: %w", err)
	}
	return &category, nil
}

// Update uses Save, not Updates, for the same zero-value reason
// documented on UserRepository.Update: IsActive=false would be
// silently dropped by Updates(struct).
func (r *categoryRepository) Update(ctx context.Context, category *entity.Category) error {
	if err := r.db.WithContext(ctx).Save(category).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return fmt.Errorf("update category: %w", entity.ErrConflict)
		}
		return fmt.Errorf("update category: %w", err)
	}
	return nil
}

// Delete soft-deletes a category and, in the same transaction,
// promotes any of its children to top-level.
//
// The migration declares `parent_id ... REFERENCES categories (id)
// ON DELETE SET NULL`, but that foreign-key action only fires on a
// real SQL DELETE — and this is a soft delete (an UPDATE setting
// deleted_at, via entity.Category's embedded gorm.DeletedAt), so the
// database-level action never triggers. Without this second update,
// a child category would keep pointing at a parent_id that no
// longer resolves to anything through the normal (non-deleted) read
// path. We replicate the FK's intent explicitly here instead.
//
// Both statements run in one transaction: either both succeed, or
// neither does. Without db.Transaction, a failure between the two
// updates (a dropped connection, a query timeout) could soft-delete
// the parent while leaving children pointing at it — the exact
// inconsistency this method exists to prevent.
func (r *categoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Delete(&entity.Category{}, "id = ?", id)
		if result.Error != nil {
			return fmt.Errorf("delete category: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("delete category: %w", entity.ErrNotFound)
		}

		if err := tx.Model(&entity.Category{}).
			Where("parent_id = ?", id).
			Update("parent_id", nil).Error; err != nil {
			return fmt.Errorf("delete category: promoting children: %w", err)
		}

		return nil
	})
}

func (r *categoryRepository) List(ctx context.Context, offset, limit int) ([]entity.Category, int64, error) {
	var categories []entity.Category
	var total int64

	if err := r.db.WithContext(ctx).Model(&entity.Category{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count categories: %w", err)
	}

	if err := r.db.WithContext(ctx).
		Order("name ASC").
		Offset(offset).
		Limit(limit).
		Find(&categories).Error; err != nil {
		return nil, 0, fmt.Errorf("list categories: %w", err)
	}

	return categories, total, nil
}
