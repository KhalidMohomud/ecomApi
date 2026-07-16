package service

import (
	"context"
	"fmt"

	"github.com/KhalidMohomud/ecomApi/internal/domain/entity"
	"github.com/KhalidMohomud/ecomApi/internal/domain/repository"
	"github.com/KhalidMohomud/ecomApi/internal/dto"
	"github.com/KhalidMohomud/ecomApi/internal/utils"
	"github.com/google/uuid"
)

type BrandService interface {
	Create(ctx context.Context, req dto.CreateBrandRequest) (*dto.BrandResponse, error)
	Update(ctx context.Context, id uuid.UUID, req dto.UpdateBrandRequest) (*dto.BrandResponse, error)
	Delete(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*dto.BrandResponse, error)
	GetBySlug(ctx context.Context, slug string) (*dto.BrandResponse, error)
	List(ctx context.Context, page, pageSize int) (*dto.PaginatedResponse[dto.BrandResponse], error)
}

type brandService struct {
	brandRepo repository.BrandRepository
}

func NewBrandService(brandRepo repository.BrandRepository) BrandService {
	return &brandService{brandRepo: brandRepo}
}

func (s *brandService) Create(ctx context.Context, req dto.CreateBrandRequest) (*dto.BrandResponse, error) {
	slug := req.Slug
	if slug == "" {
		slug = req.Name
	}
	slug = utils.Slugify(slug)

	brand := &entity.Brand{
		Name:        req.Name,
		Slug:        slug,
		Description: req.Description,
		LogoURL:     req.LogoURL,
		IsActive:    true,
	}

	if err := s.brandRepo.Create(ctx, brand); err != nil {
		return nil, fmt.Errorf("create brand: %w", err)
	}

	resp := dto.NewBrandResponse(brand)
	return &resp, nil
}

func (s *brandService) Update(ctx context.Context, id uuid.UUID, req dto.UpdateBrandRequest) (*dto.BrandResponse, error) {
	brand, err := s.brandRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("update brand: %w", err)
	}

	slug := req.Slug
	if slug == "" {
		slug = req.Name
	}
	slug = utils.Slugify(slug)

	brand.Name = req.Name
	brand.Slug = slug
	brand.Description = req.Description
	brand.LogoURL = req.LogoURL
	brand.IsActive = req.IsActive

	if err := s.brandRepo.Update(ctx, brand); err != nil {
		return nil, fmt.Errorf("update brand: %w", err)
	}

	resp := dto.NewBrandResponse(brand)
	return &resp, nil
}

func (s *brandService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.brandRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete brand: %w", err)
	}
	return nil
}

func (s *brandService) GetByID(ctx context.Context, id uuid.UUID) (*dto.BrandResponse, error) {
	brand, err := s.brandRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get brand: %w", err)
	}
	resp := dto.NewBrandResponse(brand)
	return &resp, nil
}

func (s *brandService) GetBySlug(ctx context.Context, slug string) (*dto.BrandResponse, error) {
	brand, err := s.brandRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("get brand: %w", err)
	}
	resp := dto.NewBrandResponse(brand)
	return &resp, nil
}

func (s *brandService) List(ctx context.Context, page, pageSize int) (*dto.PaginatedResponse[dto.BrandResponse], error) {
	offset := (page - 1) * pageSize

	brands, total, err := s.brandRepo.List(ctx, offset, pageSize)
	if err != nil {
		return nil, fmt.Errorf("list brands: %w", err)
	}

	items := make([]dto.BrandResponse, len(brands))
	for i, b := range brands {
		items[i] = dto.NewBrandResponse(&b)
	}

	resp := dto.NewPaginatedResponse(items, total, page, pageSize)
	return &resp, nil
}
