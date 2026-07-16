package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/KhalidMohomud/ecomApi/internal/domain/entity"
	"github.com/KhalidMohomud/ecomApi/internal/domain/repository"
	"github.com/KhalidMohomud/ecomApi/internal/dto"
	"github.com/KhalidMohomud/ecomApi/internal/utils"
	"github.com/google/uuid"
)

type CategoryService interface {
	Create(ctx context.Context, req dto.CreateCategoryRequest) (*dto.CategoryResponse, error)
	Update(ctx context.Context, id uuid.UUID, req dto.UpdateCategoryRequest) (*dto.CategoryResponse, error)
	Delete(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*dto.CategoryResponse, error)
	GetBySlug(ctx context.Context, slug string) (*dto.CategoryResponse, error)
	List(ctx context.Context, page, pageSize int) (*dto.PaginatedResponse[dto.CategoryResponse], error)
}

type categoryService struct {
	categoryRepo repository.CategoryRepository
}

func NewCategoryService(categoryRepo repository.CategoryRepository) CategoryService {
	return &categoryService{categoryRepo: categoryRepo}
}

func (s *categoryService) Create(ctx context.Context, req dto.CreateCategoryRequest) (*dto.CategoryResponse, error) {
	slug := req.Slug
	if slug == "" {
		slug = req.Name
	}
	slug = utils.Slugify(slug)

	if err := s.validateParent(ctx, req.ParentID, nil); err != nil {
		return nil, err
	}

	category := &entity.Category{
		Name:        req.Name,
		Slug:        slug,
		Description: req.Description,
		ParentID:    req.ParentID,
		IsActive:    true,
	}

	if err := s.categoryRepo.Create(ctx, category); err != nil {
		return nil, fmt.Errorf("create category: %w", err)
	}

	resp := dto.NewCategoryResponse(category)
	return &resp, nil
}

func (s *categoryService) Update(ctx context.Context, id uuid.UUID, req dto.UpdateCategoryRequest) (*dto.CategoryResponse, error) {
	category, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("update category: %w", err)
	}

	slug := req.Slug
	if slug == "" {
		slug = req.Name
	}
	slug = utils.Slugify(slug)

	if err := s.validateParent(ctx, req.ParentID, &id); err != nil {
		return nil, err
	}

	category.Name = req.Name
	category.Slug = slug
	category.Description = req.Description
	category.ParentID = req.ParentID
	category.IsActive = req.IsActive

	if err := s.categoryRepo.Update(ctx, category); err != nil {
		return nil, fmt.Errorf("update category: %w", err)
	}

	resp := dto.NewCategoryResponse(category)
	return &resp, nil
}

func (s *categoryService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.categoryRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete category: %w", err)
	}
	return nil
}

func (s *categoryService) GetByID(ctx context.Context, id uuid.UUID) (*dto.CategoryResponse, error) {
	category, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get category: %w", err)
	}
	resp := dto.NewCategoryResponse(category)
	return &resp, nil
}

func (s *categoryService) GetBySlug(ctx context.Context, slug string) (*dto.CategoryResponse, error) {
	category, err := s.categoryRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("get category: %w", err)
	}
	resp := dto.NewCategoryResponse(category)
	return &resp, nil
}

func (s *categoryService) List(ctx context.Context, page, pageSize int) (*dto.PaginatedResponse[dto.CategoryResponse], error) {
	offset := (page - 1) * pageSize

	categories, total, err := s.categoryRepo.List(ctx, offset, pageSize)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}

	items := make([]dto.CategoryResponse, len(categories))
	for i, c := range categories {
		items[i] = dto.NewCategoryResponse(&c)
	}

	resp := dto.NewPaginatedResponse(items, total, page, pageSize)
	return &resp, nil
}

// validateParent enforces two rules before a category is written:
// the parent (if any) must actually exist, and a category cannot be
// its own parent. selfID is nil on Create (there's no ID yet to
// compare against) and set on Update.
func (s *categoryService) validateParent(ctx context.Context, parentID *uuid.UUID, selfID *uuid.UUID) error {
	if parentID == nil {
		return nil
	}

	if selfID != nil && *parentID == *selfID {
		return ErrInvalidParentCategory
	}

	_, err := s.categoryRepo.GetByID(ctx, *parentID)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return ErrInvalidParentCategory
		}
		return fmt.Errorf("validate parent category: %w", err)
	}

	return nil
}
