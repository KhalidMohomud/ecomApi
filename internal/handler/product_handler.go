package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/KhalidMohomud/ecomApi/internal/domain/entity"
	"github.com/KhalidMohomud/ecomApi/internal/dto"
	"github.com/KhalidMohomud/ecomApi/internal/service"
	"github.com/KhalidMohomud/ecomApi/internal/utils"
	"github.com/KhalidMohomud/ecomApi/internal/validator"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ProductHandler follows the same public-read/admin-write split as
// CategoryHandler and BrandHandler.
type ProductHandler struct {
	productService service.ProductService
}

func NewProductHandler(productService service.ProductService) *ProductHandler {
	return &ProductHandler{productService: productService}
}

// Create godoc
//
//	@Summary		Create a product
//	@Description	Admin only. If slug is omitted, it's generated from name. price_cents is an integer — $19.99 is 1999, never a float.
//	@Tags			products
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.CreateProductRequest	true	"Product payload"
//	@Success		201		{object}	utils.SuccessResponse{data=dto.ProductResponse}
//	@Failure		400		{object}	utils.ErrorResponse	"validation failed, or category/brand does not exist"
//	@Failure		409		{object}	utils.ErrorResponse	"slug or SKU already in use"
//	@Router			/products [post]
func (h *ProductHandler) Create(c *gin.Context) {
	var req dto.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Validation failed", validator.FormatValidationErrors(err))
		return
	}

	resp, err := h.productService.Create(c.Request.Context(), req)
	if err != nil {
		h.handleProductError(c, err)
		return
	}

	utils.Success(c, http.StatusCreated, "Product created successfully", resp)
}

// Update godoc
//
//	@Summary		Update a product
//	@Description	Admin only. Full replace — every field is overwritten with the request body.
//	@Tags			products
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Product ID"
//	@Param			request	body		dto.UpdateProductRequest	true	"Product payload"
//	@Success		200		{object}	utils.SuccessResponse{data=dto.ProductResponse}
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		404		{object}	utils.ErrorResponse
//	@Router			/products/{id} [put]
func (h *ProductHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid product ID", nil)
		return
	}

	var req dto.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Validation failed", validator.FormatValidationErrors(err))
		return
	}

	resp, err := h.productService.Update(c.Request.Context(), id, req)
	if err != nil {
		h.handleProductError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Product updated successfully", resp)
}

// Delete godoc
//
//	@Summary		Delete a product
//	@Description	Admin only. Soft-deletes the product.
//	@Tags			products
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id	path		string	true	"Product ID"
//	@Success		200	{object}	utils.SuccessResponse
//	@Failure		404	{object}	utils.ErrorResponse
//	@Router			/products/{id} [delete]
func (h *ProductHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid product ID", nil)
		return
	}

	if err := h.productService.Delete(c.Request.Context(), id); err != nil {
		h.handleProductError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Product deleted successfully", nil)
}

// GetByID godoc
//
//	@Summary		Get a product by ID
//	@Tags			products
//	@Produce		json
//	@Param			id	path		string	true	"Product ID"
//	@Success		200	{object}	utils.SuccessResponse{data=dto.ProductResponse}
//	@Failure		404	{object}	utils.ErrorResponse
//	@Router			/products/{id} [get]
func (h *ProductHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid product ID", nil)
		return
	}

	resp, err := h.productService.GetByID(c.Request.Context(), id)
	if err != nil {
		h.handleProductError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Product retrieved successfully", resp)
}

// GetBySlug godoc
//
//	@Summary		Get a product by slug
//	@Tags			products
//	@Produce		json
//	@Param			slug	path		string	true	"Product slug"
//	@Success		200		{object}	utils.SuccessResponse{data=dto.ProductResponse}
//	@Failure		404		{object}	utils.ErrorResponse
//	@Router			/products/slug/{slug} [get]
func (h *ProductHandler) GetBySlug(c *gin.Context) {
	resp, err := h.productService.GetBySlug(c.Request.Context(), c.Param("slug"))
	if err != nil {
		h.handleProductError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Product retrieved successfully", resp)
}

// List godoc
//
//	@Summary		List / search / filter products
//	@Tags			products
//	@Produce		json
//	@Param			page			query		int		false	"Page number"							default(1)
//	@Param			page_size		query		int		false	"Items per page"						default(20)
//	@Param			category_id		query		string	false	"Filter by category ID"
//	@Param			brand_id		query		string	false	"Filter by brand ID"
//	@Param			min_price_cents	query		int		false	"Minimum price, in cents"
//	@Param			max_price_cents	query		int		false	"Maximum price, in cents"
//	@Param			search			query		string	false	"Case-insensitive substring match on name/description"
//	@Param			sort			query		string	false	"newest, oldest, price_asc, price_desc, name_asc, name_desc"
//	@Success		200				{object}	utils.SuccessResponse{data=dto.PaginatedResponse[dto.ProductResponse]}
//	@Failure		400				{object}	utils.ErrorResponse
//	@Router			/products [get]
func (h *ProductHandler) List(c *gin.Context) {
	var query dto.ProductListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		utils.Error(c, http.StatusBadRequest, "Validation failed", validator.FormatValidationErrors(err))
		return
	}

	resp, err := h.productService.List(c.Request.Context(), query)
	if err != nil {
		h.handleProductError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Products retrieved successfully", resp)
}

func (h *ProductHandler) handleProductError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, entity.ErrNotFound):
		utils.Error(c, http.StatusNotFound, "Product not found", nil)
	case errors.Is(err, entity.ErrConflict):
		utils.Error(c, http.StatusConflict, "A product with this slug or SKU already exists", nil)
	case errors.Is(err, service.ErrInvalidCategory):
		utils.Error(c, http.StatusBadRequest, "Invalid category", nil)
	case errors.Is(err, service.ErrInvalidBrand):
		utils.Error(c, http.StatusBadRequest, "Invalid brand", nil)
	default:
		slog.Error("product handler: unhandled error", "error", err)
		utils.Error(c, http.StatusInternalServerError, "Something went wrong", nil)
	}
}
