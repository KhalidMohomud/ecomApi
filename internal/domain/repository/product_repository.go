package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/KhalidMohomud/ecomApi/internal/domain/entity"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProductSort enumerates every sort order the API supports. It's a
// closed set (a defined string type with named constants, same
// pattern as entity.Role) specifically so that user input can never
// reach the database as a raw ORDER BY fragment — see
// productOrderClause below for why that matters.
type ProductSort string

const (
	ProductSortNewest    ProductSort = "newest"
	ProductSortOldest    ProductSort = "oldest"
	ProductSortPriceAsc  ProductSort = "price_asc"
	ProductSortPriceDesc ProductSort = "price_desc"
	ProductSortNameAsc   ProductSort = "name_asc"
	ProductSortNameDesc  ProductSort = "name_desc"
)

// ProductFilter carries every optional narrowing criterion List
// accepts. Pointer fields (CategoryID, BrandID, price bounds) are
// nil when the caller didn't specify that filter — the same "nil
// means absent" convention as entity.User.Phone — rather than using
// a zero value that could be confused with a real filter (a zero
// uuid.UUID, or a min price of 0).
type ProductFilter struct {
	CategoryID    *uuid.UUID
	BrandID       *uuid.UUID
	MinPriceCents *int64
	MaxPriceCents *int64
	Search        string
	Sort          ProductSort
}

type ProductRepository interface {
	Create(ctx context.Context, product *entity.Product) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Product, error)
	GetBySlug(ctx context.Context, slug string) (*entity.Product, error)
	Update(ctx context.Context, product *entity.Product) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter ProductFilter, offset, limit int) ([]entity.Product, int64, error)
	// ExistsByCategoryID reports whether any non-deleted product
	// still references categoryID. CategoryService.Delete calls this
	// to decide whether a category is safe to delete — see the
	// comment on that method for why this can't be left to the
	// database's ON DELETE RESTRICT alone.
	ExistsByCategoryID(ctx context.Context, categoryID uuid.UUID) (bool, error)
}

type productRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) ProductRepository {
	return &productRepository{db: db}
}

func (r *productRepository) Create(ctx context.Context, product *entity.Product) error {
	if err := r.db.WithContext(ctx).Create(product).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return fmt.Errorf("create product: %w", entity.ErrConflict)
		}
		return fmt.Errorf("create product: %w", err)
	}
	return nil
}

func (r *productRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Product, error) {
	var product entity.Product
	if err := r.db.WithContext(ctx).First(&product, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("get product by id: %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("get product by id: %w", err)
	}
	return &product, nil
}

func (r *productRepository) GetBySlug(ctx context.Context, slug string) (*entity.Product, error) {
	var product entity.Product
	if err := r.db.WithContext(ctx).First(&product, "slug = ?", slug).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("get product by slug: %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("get product by slug: %w", err)
	}
	return &product, nil
}

func (r *productRepository) Update(ctx context.Context, product *entity.Product) error {
	if err := r.db.WithContext(ctx).Save(product).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return fmt.Errorf("update product: %w", entity.ErrConflict)
		}
		return fmt.Errorf("update product: %w", err)
	}
	return nil
}

func (r *productRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&entity.Product{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete product: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("delete product: %w", entity.ErrNotFound)
	}
	return nil
}

// List runs the count and the page fetch as two independent query
// chains, both built from applyProductFilters, rather than reusing
// one *gorm.DB across Count() then Find(). GORM's statement building
// isn't guaranteed to behave the way you'd expect if you keep
// chaining onto a *gorm.DB after Count() has already executed —
// starting fresh each time sidesteps that entirely. This is the same
// two-chain shape every other repository's List method already uses
// (see UserRepository.List); here it just needs a shared filter
// function so the WHERE clauses aren't duplicated by hand twice.
func (r *productRepository) List(ctx context.Context, filter ProductFilter, offset, limit int) ([]entity.Product, int64, error) {
	var total int64
	countQuery := applyProductFilters(r.db.WithContext(ctx).Model(&entity.Product{}), filter)
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count products: %w", err)
	}

	var products []entity.Product
	findQuery := applyProductFilters(r.db.WithContext(ctx).Model(&entity.Product{}), filter)
	if err := findQuery.
		Order(productOrderClause(filter.Sort)).
		Offset(offset).
		Limit(limit).
		Find(&products).Error; err != nil {
		return nil, 0, fmt.Errorf("list products: %w", err)
	}

	return products, total, nil
}

func applyProductFilters(query *gorm.DB, filter ProductFilter) *gorm.DB {
	if filter.CategoryID != nil {
		query = query.Where("category_id = ?", *filter.CategoryID)
	}
	if filter.BrandID != nil {
		query = query.Where("brand_id = ?", *filter.BrandID)
	}
	if filter.MinPriceCents != nil {
		query = query.Where("price_cents >= ?", *filter.MinPriceCents)
	}
	if filter.MaxPriceCents != nil {
		query = query.Where("price_cents <= ?", *filter.MaxPriceCents)
	}
	if filter.Search != "" {
		// ILIKE is Postgres's case-insensitive LIKE. A leading and
		// trailing "%" makes it a substring match. This is a simple,
		// correct approach that will happily serve a catalog of
		// thousands of products; at a much larger scale you'd reach
		// for a proper full-text index (tsvector + GIN, or pg_trgm)
		// instead, since ILIKE '%term%' can't use a normal B-tree
		// index and forces a sequential scan. Not needed yet — this
		// is the kind of thing you optimize when you have the traffic
		// that justifies it, not preemptively.
		like := "%" + filter.Search + "%"
		query = query.Where("name ILIKE ? OR description ILIKE ?", like, like)
	}
	return query
}

// productOrderClause maps a ProductSort to a hardcoded ORDER BY
// fragment. This function NEVER interpolates filter.Sort (or any
// other user-controlled string) directly into SQL — it switches on
// the value and returns one of a fixed set of literal strings.
//
// Why this matters: if List instead did something like
// `.Order(string(filter.Sort))`, a client could set ?sort=anything
// and have it dropped verbatim into the query's ORDER BY clause.
// ORDER BY doesn't take placeholder parameters the way WHERE does,
// so this is a genuine SQL injection vector many tutorials miss —
// "user input in the sort field" doesn't feel as dangerous as "user
// input in a WHERE clause," but it's the exact same class of bug.
// Binding-level validation (`binding:"omitempty,oneof=..."` on the
// DTO) is a second layer of defense, not a substitute for this one.
func productOrderClause(sort ProductSort) string {
	switch sort {
	case ProductSortOldest:
		return "created_at ASC"
	case ProductSortPriceAsc:
		return "price_cents ASC"
	case ProductSortPriceDesc:
		return "price_cents DESC"
	case ProductSortNameAsc:
		return "name ASC"
	case ProductSortNameDesc:
		return "name DESC"
	case ProductSortNewest:
		return "created_at DESC"
	default:
		return "created_at DESC"
	}
}

func (r *productRepository) ExistsByCategoryID(ctx context.Context, categoryID uuid.UUID) (bool, error) {
	var count int64
	// Deliberately a plain Count, not Count combined with Limit(1) —
	// GORM's handling of Limit alongside Count has version-dependent
	// quirks (whether it wraps the query in a subquery to actually
	// respect the limit). category_id is indexed, so a full count is
	// already fast; simple and correct beats a micro-optimization
	// that risks silently counting wrong.
	if err := r.db.WithContext(ctx).
		Model(&entity.Product{}).
		Where("category_id = ?", categoryID).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("check products by category: %w", err)
	}
	return count > 0, nil
}
