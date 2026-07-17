package dto

import (
	"time"

	"github.com/KhalidMohomud/ecomApi/internal/domain/entity"
	"github.com/google/uuid"
)

// CreateProductRequest takes PriceCents, not a float Price. The
// client is responsible for converting a displayed "$19.99" into
// 1999 before sending it, the same way the response hands back 1999
// for the client to format however its UI needs. Accepting a float
// here would just move the rounding-error risk from storage to the
// request boundary instead of eliminating it.
//
// Unlike Slug (auto-generated from Name if omitted, same as
// Category/Brand), SKU has no auto-generation — SKUs are usually
// assigned deliberately by whoever manages inventory, often
// following a structured scheme (warehouse code, category prefix,
// etc.) that a slugified product name can't replicate.
type CreateProductRequest struct {
	Name        string     `json:"name" binding:"required,min=2,max=200"`
	Slug        string     `json:"slug" binding:"omitempty,max=220"`
	SKU         string     `json:"sku" binding:"required,min=1,max=64"`
	Description string     `json:"description" binding:"omitempty,max=5000"`
	PriceCents  int64      `json:"price_cents" binding:"required,gt=0"`
	CategoryID  uuid.UUID  `json:"category_id" binding:"required"`
	BrandID     *uuid.UUID `json:"brand_id"`
}

type UpdateProductRequest struct {
	Name        string     `json:"name" binding:"required,min=2,max=200"`
	Slug        string     `json:"slug" binding:"omitempty,max=220"`
	SKU         string     `json:"sku" binding:"required,min=1,max=64"`
	Description string     `json:"description" binding:"omitempty,max=5000"`
	PriceCents  int64      `json:"price_cents" binding:"required,gt=0"`
	CategoryID  uuid.UUID  `json:"category_id" binding:"required"`
	BrandID     *uuid.UUID `json:"brand_id"`
	IsActive    bool       `json:"is_active"`
}

// ProductListQuery is bound from query parameters on GET
// /products — it embeds PaginationQuery (page, page_size) the same
// way every other list endpoint does, adding the filter/search/sort
// parameters specific to products.
//
// CategoryID and BrandID are plain strings here, not *uuid.UUID.
// Gin's query/form binding only converts into primitive kinds
// (string, the numeric types, bool) — it does not know how to
// populate a custom struct type like uuid.UUID from a query string,
// even though uuid.UUID implements encoding.TextUnmarshaler and JSON
// body binding (c.ShouldBindJSON, used everywhere else in this
// project) handles that case fine. The `uuid` validator tag still
// gives us the same 400-on-malformed-input behavior; the actual
// string-to-uuid.UUID conversion happens in ProductService.List,
// right before building the repository filter.
//
// Sort uses `oneof` to reject anything outside the fixed set of
// values before it ever reaches the service or repository layer.
// That's a second line of defense on top of (not a replacement for)
// productOrderClause's hardcoded switch in the repository — see the
// comment there for why user input must never reach an ORDER BY
// clause directly.
type ProductListQuery struct {
	PaginationQuery

	CategoryID    string `form:"category_id" binding:"omitempty,uuid"`
	BrandID       string `form:"brand_id" binding:"omitempty,uuid"`
	MinPriceCents *int64 `form:"min_price_cents" binding:"omitempty,min=0"`
	MaxPriceCents *int64     `form:"max_price_cents" binding:"omitempty,min=0"`
	Search        string     `form:"search" binding:"omitempty,max=200"`
	Sort          string     `form:"sort" binding:"omitempty,oneof=newest oldest price_asc price_desc name_asc name_desc"`
}

type ProductResponse struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	SKU         string     `json:"sku"`
	Description string     `json:"description"`

	// PriceCents is the authoritative value — exactly what's stored,
	// safe to do further arithmetic on. Price is a derived
	// convenience for clients that just want to display a number;
	// it's computed fresh on every response, never stored or parsed
	// from a request.
	PriceCents int64   `json:"price_cents"`
	Price      float64 `json:"price"`

	CategoryID uuid.UUID  `json:"category_id"`
	BrandID    *uuid.UUID `json:"brand_id,omitempty"`
	IsActive   bool       `json:"is_active"`
	CreatedAt  time.Time  `json:"created_at"`
}

func NewProductResponse(p *entity.Product) ProductResponse {
	return ProductResponse{
		ID:          p.ID,
		Name:        p.Name,
		Slug:        p.Slug,
		SKU:         p.SKU,
		Description: p.Description,
		PriceCents:  p.PriceCents,
		Price:       float64(p.PriceCents) / 100,
		CategoryID:  p.CategoryID,
		BrandID:     p.BrandID,
		IsActive:    p.IsActive,
		CreatedAt:   p.CreatedAt,
	}
}
