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

// BrandHandler follows the same public-read/admin-write split as
// CategoryHandler — see the comment there for the routing rationale.
type BrandHandler struct {
	brandService service.BrandService
}

func NewBrandHandler(brandService service.BrandService) *BrandHandler {
	return &BrandHandler{brandService: brandService}
}

// Create godoc
//
//	@Summary		Create a brand
//	@Description	Admin only. If slug is omitted, it's generated from name.
//	@Tags			brands
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.CreateBrandRequest	true	"Brand payload"
//	@Success		201		{object}	utils.SuccessResponse{data=dto.BrandResponse}
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		409		{object}	utils.ErrorResponse	"slug already in use"
//	@Router			/brands [post]
func (h *BrandHandler) Create(c *gin.Context) {
	var req dto.CreateBrandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Validation failed", validator.FormatValidationErrors(err))
		return
	}

	resp, err := h.brandService.Create(c.Request.Context(), req)
	if err != nil {
		h.handleBrandError(c, err)
		return
	}

	utils.Success(c, http.StatusCreated, "Brand created successfully", resp)
}

// Update godoc
//
//	@Summary		Update a brand
//	@Description	Admin only. Full replace — every field is overwritten with the request body.
//	@Tags			brands
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Brand ID"
//	@Param			request	body		dto.UpdateBrandRequest	true	"Brand payload"
//	@Success		200		{object}	utils.SuccessResponse{data=dto.BrandResponse}
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		404		{object}	utils.ErrorResponse
//	@Router			/brands/{id} [put]
func (h *BrandHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid brand ID", nil)
		return
	}

	var req dto.UpdateBrandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Validation failed", validator.FormatValidationErrors(err))
		return
	}

	resp, err := h.brandService.Update(c.Request.Context(), id, req)
	if err != nil {
		h.handleBrandError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Brand updated successfully", resp)
}

// Delete godoc
//
//	@Summary		Delete a brand
//	@Description	Admin only. Soft-deletes the brand.
//	@Tags			brands
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id	path		string	true	"Brand ID"
//	@Success		200	{object}	utils.SuccessResponse
//	@Failure		404	{object}	utils.ErrorResponse
//	@Router			/brands/{id} [delete]
func (h *BrandHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid brand ID", nil)
		return
	}

	if err := h.brandService.Delete(c.Request.Context(), id); err != nil {
		h.handleBrandError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Brand deleted successfully", nil)
}

// GetByID godoc
//
//	@Summary		Get a brand by ID
//	@Tags			brands
//	@Produce		json
//	@Param			id	path		string	true	"Brand ID"
//	@Success		200	{object}	utils.SuccessResponse{data=dto.BrandResponse}
//	@Failure		404	{object}	utils.ErrorResponse
//	@Router			/brands/{id} [get]
func (h *BrandHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid brand ID", nil)
		return
	}

	resp, err := h.brandService.GetByID(c.Request.Context(), id)
	if err != nil {
		h.handleBrandError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Brand retrieved successfully", resp)
}

// GetBySlug godoc
//
//	@Summary		Get a brand by slug
//	@Tags			brands
//	@Produce		json
//	@Param			slug	path		string	true	"Brand slug"
//	@Success		200		{object}	utils.SuccessResponse{data=dto.BrandResponse}
//	@Failure		404		{object}	utils.ErrorResponse
//	@Router			/brands/slug/{slug} [get]
func (h *BrandHandler) GetBySlug(c *gin.Context) {
	resp, err := h.brandService.GetBySlug(c.Request.Context(), c.Param("slug"))
	if err != nil {
		h.handleBrandError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Brand retrieved successfully", resp)
}

// List godoc
//
//	@Summary		List brands
//	@Tags			brands
//	@Produce		json
//	@Param			page		query		int	false	"Page number"		default(1)
//	@Param			page_size	query		int	false	"Items per page"	default(20)
//	@Success		200			{object}	utils.SuccessResponse{data=dto.PaginatedResponse[dto.BrandResponse]}
//	@Router			/brands [get]
func (h *BrandHandler) List(c *gin.Context) {
	var query dto.PaginationQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid pagination parameters", nil)
		return
	}

	resp, err := h.brandService.List(c.Request.Context(), query.Page, query.PageSize)
	if err != nil {
		h.handleBrandError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Brands retrieved successfully", resp)
}

func (h *BrandHandler) handleBrandError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, entity.ErrNotFound):
		utils.Error(c, http.StatusNotFound, "Brand not found", nil)
	case errors.Is(err, entity.ErrConflict):
		utils.Error(c, http.StatusConflict, "A brand with this slug already exists", nil)
	default:
		slog.Error("brand handler: unhandled error", "error", err)
		utils.Error(c, http.StatusInternalServerError, "Something went wrong", nil)
	}
}
