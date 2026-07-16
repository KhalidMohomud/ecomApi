package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/KhalidMohomud/ecomApi/internal/domain/entity"
	"github.com/KhalidMohomud/ecomApi/internal/dto"
	"github.com/KhalidMohomud/ecomApi/internal/middleware"
	"github.com/KhalidMohomud/ecomApi/internal/service"
	"github.com/KhalidMohomud/ecomApi/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AdminHandler serves admin-only endpoints. Every route this handler
// serves is mounted behind both middleware.Auth and
// middleware.RequireRole(entity.RoleAdmin) in routes.go — this
// handler itself does not re-check the role, the same way UserHandler
// doesn't re-check that a token exists. Authorization is the router's
// job to enforce and this file's job to trust.
type AdminHandler struct {
	adminService service.AdminService
}

func NewAdminHandler(adminService service.AdminService) *AdminHandler {
	return &AdminHandler{adminService: adminService}
}

// GetDashboard godoc
//
//	@Summary		Admin dashboard summary
//	@Tags			admin
//	@Security		BearerAuth
//	@Produce		json
//	@Success		200	{object}	utils.SuccessResponse{data=dto.DashboardResponse}
//	@Failure		401	{object}	utils.ErrorResponse
//	@Failure		403	{object}	utils.ErrorResponse	"caller is not an admin"
//	@Router			/admin/dashboard [get]
func (h *AdminHandler) GetDashboard(c *gin.Context) {
	resp, err := h.adminService.GetDashboard(c.Request.Context())
	if err != nil {
		h.handleAdminError(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Dashboard retrieved successfully", resp)
}

// ListUsers godoc
//
//	@Summary		List all users
//	@Tags			admin
//	@Security		BearerAuth
//	@Produce		json
//	@Param			page		query		int	false	"Page number"		default(1)
//	@Param			page_size	query		int	false	"Items per page"	default(20)
//	@Success		200			{object}	utils.SuccessResponse{data=dto.PaginatedResponse[dto.UserResponse]}
//	@Failure		401			{object}	utils.ErrorResponse
//	@Failure		403			{object}	utils.ErrorResponse
//	@Router			/admin/users [get]
func (h *AdminHandler) ListUsers(c *gin.Context) {
	var query dto.PaginationQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid pagination parameters", nil)
		return
	}

	resp, err := h.adminService.ListUsers(c.Request.Context(), query.Page, query.PageSize)
	if err != nil {
		h.handleAdminError(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Users retrieved successfully", resp)
}

// BlockUser godoc
//
//	@Summary		Block a user
//	@Description	Deactivates the account and revokes all its refresh tokens. An access token already issued to that user stays valid until it expires (up to 15 minutes) — see the code comment on AdminService.BlockUser for why.
//	@Tags			admin
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id	path		string	true	"User ID"
//	@Success		200	{object}	utils.SuccessResponse
//	@Failure		400	{object}	utils.ErrorResponse	"cannot block your own account"
//	@Failure		404	{object}	utils.ErrorResponse
//	@Router			/admin/users/{id}/block [post]
func (h *AdminHandler) BlockUser(c *gin.Context) {
	h.actOnUser(c, h.adminService.BlockUser, "User blocked successfully")
}

// UnblockUser godoc
//
//	@Summary		Unblock a user
//	@Tags			admin
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id	path		string	true	"User ID"
//	@Success		200	{object}	utils.SuccessResponse
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		404	{object}	utils.ErrorResponse
//	@Router			/admin/users/{id}/unblock [post]
func (h *AdminHandler) UnblockUser(c *gin.Context) {
	h.actOnUser(c, h.adminService.UnblockUser, "User unblocked successfully")
}

// DeleteUser godoc
//
//	@Summary		Delete a user
//	@Description	Soft-deletes the account and revokes all its refresh tokens.
//	@Tags			admin
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id	path		string	true	"User ID"
//	@Success		200	{object}	utils.SuccessResponse
//	@Failure		400	{object}	utils.ErrorResponse	"cannot delete your own account"
//	@Failure		404	{object}	utils.ErrorResponse
//	@Router			/admin/users/{id} [delete]
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	h.actOnUser(c, h.adminService.DeleteUser, "User deleted successfully")
}

// actOnUser is the shape shared by Block/Unblock/Delete: parse the
// target ID out of the URL path, read the acting admin's ID (set by
// middleware.Auth) off the context, call one service method, respond.
// action's signature — func(ctx, actorID, targetID) error — matches
// AdminService.BlockUser/UnblockUser/DeleteUser exactly, so each
// caller above passes the method itself as a value instead of
// wrapping it in a closure.
func (h *AdminHandler) actOnUser(c *gin.Context, action func(ctx context.Context, actorID, targetID uuid.UUID) error, successMessage string) {
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid user ID", nil)
		return
	}

	actorID, ok := middleware.UserIDFromContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	if err := action(c.Request.Context(), actorID, targetID); err != nil {
		h.handleAdminError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, successMessage, nil)
}

func (h *AdminHandler) handleAdminError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, entity.ErrNotFound):
		utils.Error(c, http.StatusNotFound, "User not found", nil)
	case errors.Is(err, service.ErrCannotModifySelf):
		utils.Error(c, http.StatusBadRequest, "You cannot perform this action on your own account", nil)
	default:
		slog.Error("admin handler: unhandled error", "error", err)
		utils.Error(c, http.StatusInternalServerError, "Something went wrong", nil)
	}
}
