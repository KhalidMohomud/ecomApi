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

// CategoryHandler mixes public and admin-only routes on the same
// resource path (/categories) — unlike AdminHandler, where every
// method sits behind one group-level middleware.Use, routes.go
// applies middleware.Auth + middleware.RequireRole per-route here,
// only on Create/Update/Delete. GetByID/GetBySlug/List stay public:
// browsing a storefront's categories shouldn't require an account.
type CategoryHandler struct {
	categoryService service.CategoryService
}

func NewCategoryHandler(categoryService service.CategoryService) *CategoryHandler {
	return &CategoryHandler{categoryService: categoryService}
}

// Create godoc
//
//	@Summary		Create a category
//	@Description	Admin only. If slug is omitted, it's generated from name.
//	@Tags			categories
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.CreateCategoryRequest	true	"Category payload"
//	@Success		201		{object}	utils.SuccessResponse{data=dto.CategoryResponse}
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		409		{object}	utils.ErrorResponse	"slug already in use"
//	@Router			/categories [post]
func (h *CategoryHandler) Create(c *gin.Context) {
	var req dto.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Validation failed", validator.FormatValidationErrors(err))
		return
	}

	resp, err := h.categoryService.Create(c.Request.Context(), req)
	if err != nil {
		h.handleCategoryError(c, err)
		return
	}

	utils.Success(c, http.StatusCreated, "Category created successfully", resp)
}

// Update godoc
//
//	@Summary		Update a category
//	@Description	Admin only. Full replace — every field is overwritten with the request body.
//	@Tags			categories
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Category ID"
//	@Param			request	body		dto.UpdateCategoryRequest	true	"Category payload"
//	@Success		200		{object}	utils.SuccessResponse{data=dto.CategoryResponse}
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		404		{object}	utils.ErrorResponse
//	@Router			/categories/{id} [put]
func (h *CategoryHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid category ID", nil)
		return
	}

	var req dto.UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Validation failed", validator.FormatValidationErrors(err))
		return
	}

	resp, err := h.categoryService.Update(c.Request.Context(), id, req)
	if err != nil {
		h.handleCategoryError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Category updated successfully", resp)
}

// Delete godoc
//
//	@Summary		Delete a category
//	@Description	Admin only. Soft-deletes the category; any child categories are promoted to top-level.
//	@Tags			categories
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id	path		string	true	"Category ID"
//	@Success		200	{object}	utils.SuccessResponse
//	@Failure		404	{object}	utils.ErrorResponse
//	@Router			/categories/{id} [delete]
func (h *CategoryHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid category ID", nil)
		return
	}

	if err := h.categoryService.Delete(c.Request.Context(), id); err != nil {
		h.handleCategoryError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Category deleted successfully", nil)
}

// GetByID godoc
//
//	@Summary		Get a category by ID
//	@Tags			categories
//	@Produce		json
//	@Param			id	path		string	true	"Category ID"
//	@Success		200	{object}	utils.SuccessResponse{data=dto.CategoryResponse}
//	@Failure		404	{object}	utils.ErrorResponse
//	@Router			/categories/{id} [get]
func (h *CategoryHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid category ID", nil)
		return
	}

	resp, err := h.categoryService.GetByID(c.Request.Context(), id)
	if err != nil {
		h.handleCategoryError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Category retrieved successfully", resp)
}

// GetBySlug godoc
//
//	@Summary		Get a category by slug
//	@Tags			categories
//	@Produce		json
//	@Param			slug	path		string	true	"Category slug"
//	@Success		200		{object}	utils.SuccessResponse{data=dto.CategoryResponse}
//	@Failure		404		{object}	utils.ErrorResponse
//	@Router			/categories/slug/{slug} [get]
func (h *CategoryHandler) GetBySlug(c *gin.Context) {
	resp, err := h.categoryService.GetBySlug(c.Request.Context(), c.Param("slug"))
	if err != nil {
		h.handleCategoryError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Category retrieved successfully", resp)
}

// List godoc
//
//	@Summary		List categories
//	@Tags			categories
//	@Produce		json
//	@Param			page		query		int	false	"Page number"		default(1)
//	@Param			page_size	query		int	false	"Items per page"	default(20)
//	@Success		200			{object}	utils.SuccessResponse{data=dto.PaginatedResponse[dto.CategoryResponse]}
//	@Router			/categories [get]
func (h *CategoryHandler) List(c *gin.Context) {
	var query dto.PaginationQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid pagination parameters", nil)
		return
	}

	resp, err := h.categoryService.List(c.Request.Context(), query.Page, query.PageSize)
	if err != nil {
		h.handleCategoryError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Categories retrieved successfully", resp)
}

func (h *CategoryHandler) handleCategoryError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, entity.ErrNotFound):
		utils.Error(c, http.StatusNotFound, "Category not found", nil)
	case errors.Is(err, entity.ErrConflict):
		utils.Error(c, http.StatusConflict, "A category with this slug already exists", nil)
	case errors.Is(err, service.ErrInvalidParentCategory):
		utils.Error(c, http.StatusBadRequest, "Invalid parent category", nil)
	case errors.Is(err, service.ErrCategoryHasProducts):
		utils.Error(c, http.StatusBadRequest, "Category still has products assigned to it", nil)
	default:
		slog.Error("category handler: unhandled error", "error", err)
		utils.Error(c, http.StatusInternalServerError, "Something went wrong", nil)
	}
}
