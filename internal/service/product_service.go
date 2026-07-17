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

type ProductService interface {
	Create(ctx context.Context, req dto.CreateProductRequest) (*dto.ProductResponse, error)
	Update(ctx context.Context, id uuid.UUID, req dto.UpdateProductRequest) (*dto.ProductResponse, error)
	Delete(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*dto.ProductResponse, error)
	GetBySlug(ctx context.Context, slug string) (*dto.ProductResponse, error)
	List(ctx context.Context, query dto.ProductListQuery) (*dto.PaginatedResponse[dto.ProductResponse], error)
}

type productService struct {
	productRepo  repository.ProductRepository
	categoryRepo repository.CategoryRepository
	brandRepo    repository.BrandRepository
}

func NewProductService(
	productRepo repository.ProductRepository,
	categoryRepo repository.CategoryRepository,
	brandRepo repository.BrandRepository,
) ProductService {
	return &productService{productRepo: productRepo, categoryRepo: categoryRepo, brandRepo: brandRepo}
}

func (s *productService) Create(ctx context.Context, req dto.CreateProductRequest) (*dto.ProductResponse, error) {
	if err := s.validateReferences(ctx, req.CategoryID, req.BrandID); err != nil {
		return nil, err
	}

	slug := req.Slug
	if slug == "" {
		slug = req.Name
	}
	slug = utils.Slugify(slug)

	product := &entity.Product{
		Name:        req.Name,
		Slug:        slug,
		SKU:         req.SKU,
		Description: req.Description,
		PriceCents:  req.PriceCents,
		CategoryID:  req.CategoryID,
		BrandID:     req.BrandID,
		IsActive:    true,
	}

	if err := s.productRepo.Create(ctx, product); err != nil {
		return nil, fmt.Errorf("create product: %w", err)
	}

	resp := dto.NewProductResponse(product)
	return &resp, nil
}

func (s *productService) Update(ctx context.Context, id uuid.UUID, req dto.UpdateProductRequest) (*dto.ProductResponse, error) {
	product, err := s.productRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("update product: %w", err)
	}

	if err := s.validateReferences(ctx, req.CategoryID, req.BrandID); err != nil {
		return nil, err
	}

	slug := req.Slug
	if slug == "" {
		slug = req.Name
	}
	slug = utils.Slugify(slug)

	product.Name = req.Name
	product.Slug = slug
	product.SKU = req.SKU
	product.Description = req.Description
	product.PriceCents = req.PriceCents
	product.CategoryID = req.CategoryID
	product.BrandID = req.BrandID
	product.IsActive = req.IsActive

	if err := s.productRepo.Update(ctx, product); err != nil {
		return nil, fmt.Errorf("update product: %w", err)
	}

	resp := dto.NewProductResponse(product)
	return &resp, nil
}

func (s *productService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.productRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	return nil
}

func (s *productService) GetByID(ctx context.Context, id uuid.UUID) (*dto.ProductResponse, error) {
	product, err := s.productRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get product: %w", err)
	}
	resp := dto.NewProductResponse(product)
	return &resp, nil
}

func (s *productService) GetBySlug(ctx context.Context, slug string) (*dto.ProductResponse, error) {
	product, err := s.productRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("get product: %w", err)
	}
	resp := dto.NewProductResponse(product)
	return &resp, nil
}

func (s *productService) List(ctx context.Context, query dto.ProductListQuery) (*dto.PaginatedResponse[dto.ProductResponse], error) {
	filter := repository.ProductFilter{
		MinPriceCents: query.MinPriceCents,
		MaxPriceCents: query.MaxPriceCents,
		Search:        query.Search,
		Sort:          repository.ProductSort(query.Sort),
	}

	// query.CategoryID/BrandID are already known-valid UUID strings
	// (or empty) by the time they get here — the `uuid` binding tag
	// on dto.ProductListQuery rejected anything malformed before the
	// handler ever called this method. uuid.Parse can't meaningfully
	// fail below; it's still checked rather than using MustParse
	// because "can't happen" is a claim about today's callers, not a
	// guarantee the compiler enforces.
	if query.CategoryID != "" {
		id, err := uuid.Parse(query.CategoryID)
		if err != nil {
			return nil, fmt.Errorf("list products: parsing category_id: %w", err)
		}
		filter.CategoryID = &id
	}
	if query.BrandID != "" {
		id, err := uuid.Parse(query.BrandID)
		if err != nil {
			return nil, fmt.Errorf("list products: parsing brand_id: %w", err)
		}
		filter.BrandID = &id
	}

	products, total, err := s.productRepo.List(ctx, filter, query.Offset(), query.PageSize)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}

	items := make([]dto.ProductResponse, len(products))
	for i, p := range products {
		items[i] = dto.NewProductResponse(&p)
	}

	resp := dto.NewPaginatedResponse(items, total, query.Page, query.PageSize)
	return &resp, nil
}

// validateReferences confirms category_id points at a real category
// (required) and, if brand_id was supplied, that it points at a real
// brand too (optional — nil is always valid). Same "check before you
// write" pattern as CategoryService.validateParent, applied to a
// product's two foreign keys.
func (s *productService) validateReferences(ctx context.Context, categoryID uuid.UUID, brandID *uuid.UUID) error {
	if _, err := s.categoryRepo.GetByID(ctx, categoryID); err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return ErrInvalidCategory
		}
		return fmt.Errorf("validate category: %w", err)
	}

	if brandID != nil {
		if _, err := s.brandRepo.GetByID(ctx, *brandID); err != nil {
			if errors.Is(err, entity.ErrNotFound) {
				return ErrInvalidBrand
			}
			return fmt.Errorf("validate brand: %w", err)
		}
	}

	return nil
}
