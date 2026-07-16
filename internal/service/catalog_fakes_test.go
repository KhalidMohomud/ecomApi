package service_test

import (
	"context"
	"fmt"
	"time"

	"github.com/KhalidMohomud/ecomApi/internal/domain/entity"
	"github.com/google/uuid"
)

// Same in-memory-map fake pattern as fakeUserRepository in
// auth_service_test.go, applied to Category. By the third repository
// interface, writing one of these should feel mechanical — that's
// the point of having a consistent repository shape.
type fakeCategoryRepository struct {
	byID   map[uuid.UUID]*entity.Category
	bySlug map[string]*entity.Category
}

func newFakeCategoryRepository() *fakeCategoryRepository {
	return &fakeCategoryRepository{
		byID:   make(map[uuid.UUID]*entity.Category),
		bySlug: make(map[string]*entity.Category),
	}
}

func (f *fakeCategoryRepository) Create(_ context.Context, c *entity.Category) error {
	if _, exists := f.bySlug[c.Slug]; exists {
		return fmt.Errorf("create category: %w", entity.ErrConflict)
	}
	c.ID = uuid.New()
	c.CreatedAt = time.Now()
	f.byID[c.ID] = c
	f.bySlug[c.Slug] = c
	return nil
}

func (f *fakeCategoryRepository) GetByID(_ context.Context, id uuid.UUID) (*entity.Category, error) {
	c, ok := f.byID[id]
	if !ok {
		return nil, fmt.Errorf("get category by id: %w", entity.ErrNotFound)
	}
	return c, nil
}

func (f *fakeCategoryRepository) GetBySlug(_ context.Context, slug string) (*entity.Category, error) {
	c, ok := f.bySlug[slug]
	if !ok {
		return nil, fmt.Errorf("get category by slug: %w", entity.ErrNotFound)
	}
	return c, nil
}

func (f *fakeCategoryRepository) Update(_ context.Context, c *entity.Category) error {
	f.byID[c.ID] = c
	f.bySlug[c.Slug] = c
	return nil
}

func (f *fakeCategoryRepository) Delete(_ context.Context, id uuid.UUID) error {
	c, ok := f.byID[id]
	if !ok {
		return fmt.Errorf("delete category: %w", entity.ErrNotFound)
	}
	delete(f.byID, id)
	delete(f.bySlug, c.Slug)

	// Mirrors the "promote children to top-level" behavior
	// CategoryRepository.Delete implements via a real transaction —
	// see the comment there for why this can't be left to a database
	// FK action alone.
	for _, child := range f.byID {
		if child.ParentID != nil && *child.ParentID == id {
			child.ParentID = nil
		}
	}
	return nil
}

func (f *fakeCategoryRepository) List(_ context.Context, offset, limit int) ([]entity.Category, int64, error) {
	total := int64(len(f.byID))
	all := make([]entity.Category, 0, len(f.byID))
	for _, c := range f.byID {
		all = append(all, *c)
	}
	if offset >= len(all) {
		return []entity.Category{}, total, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], total, nil
}

type fakeBrandRepository struct {
	byID   map[uuid.UUID]*entity.Brand
	bySlug map[string]*entity.Brand
}

func newFakeBrandRepository() *fakeBrandRepository {
	return &fakeBrandRepository{
		byID:   make(map[uuid.UUID]*entity.Brand),
		bySlug: make(map[string]*entity.Brand),
	}
}

func (f *fakeBrandRepository) Create(_ context.Context, b *entity.Brand) error {
	if _, exists := f.bySlug[b.Slug]; exists {
		return fmt.Errorf("create brand: %w", entity.ErrConflict)
	}
	b.ID = uuid.New()
	b.CreatedAt = time.Now()
	f.byID[b.ID] = b
	f.bySlug[b.Slug] = b
	return nil
}

func (f *fakeBrandRepository) GetByID(_ context.Context, id uuid.UUID) (*entity.Brand, error) {
	b, ok := f.byID[id]
	if !ok {
		return nil, fmt.Errorf("get brand by id: %w", entity.ErrNotFound)
	}
	return b, nil
}

func (f *fakeBrandRepository) GetBySlug(_ context.Context, slug string) (*entity.Brand, error) {
	b, ok := f.bySlug[slug]
	if !ok {
		return nil, fmt.Errorf("get brand by slug: %w", entity.ErrNotFound)
	}
	return b, nil
}

func (f *fakeBrandRepository) Update(_ context.Context, b *entity.Brand) error {
	f.byID[b.ID] = b
	f.bySlug[b.Slug] = b
	return nil
}

func (f *fakeBrandRepository) Delete(_ context.Context, id uuid.UUID) error {
	b, ok := f.byID[id]
	if !ok {
		return fmt.Errorf("delete brand: %w", entity.ErrNotFound)
	}
	delete(f.byID, id)
	delete(f.bySlug, b.Slug)
	return nil
}

func (f *fakeBrandRepository) List(_ context.Context, offset, limit int) ([]entity.Brand, int64, error) {
	total := int64(len(f.byID))
	all := make([]entity.Brand, 0, len(f.byID))
	for _, b := range f.byID {
		all = append(all, *b)
	}
	if offset >= len(all) {
		return []entity.Brand{}, total, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], total, nil
}
