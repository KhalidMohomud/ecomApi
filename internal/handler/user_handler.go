package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/KhalidMohomud/ecomApi/internal/domain/entity"
	"github.com/KhalidMohomud/ecomApi/internal/dto"
	"github.com/KhalidMohomud/ecomApi/internal/middleware"
	"github.com/KhalidMohomud/ecomApi/internal/service"
	"github.com/KhalidMohomud/ecomApi/internal/utils"
	"github.com/KhalidMohomud/ecomApi/internal/validator"
	"github.com/gin-gonic/gin"
)

// UserHandler serves the self-service "my account" endpoints. Every
// method here assumes middleware.Auth already ran — routes.go is
// responsible for only ever mounting these behind that middleware —
// so UserIDFromContext is always expected to succeed.
type UserHandler struct {
	userService service.UserService
}

func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// GetProfile godoc
//
//	@Summary		Get the current user's profile
//	@Tags			users
//	@Security		BearerAuth
//	@Produce		json
//	@Success		200	{object}	utils.SuccessResponse{data=dto.UserResponse}
//	@Failure		401	{object}	utils.ErrorResponse
//	@Router			/users/me [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	resp, err := h.userService.GetProfile(c.Request.Context(), userID)
	if err != nil {
		h.handleUserError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Profile retrieved successfully", resp)
}

// UpdateProfile godoc
//
//	@Summary		Update the current user's profile
//	@Description	Updates name and phone only — email and role cannot be changed through this endpoint.
//	@Tags			users
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.UpdateProfileRequest	true	"Profile fields to update"
//	@Success		200		{object}	utils.SuccessResponse{data=dto.UserResponse}
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		401		{object}	utils.ErrorResponse
//	@Router			/users/me [put]
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Validation failed", validator.FormatValidationErrors(err))
		return
	}

	resp, err := h.userService.UpdateProfile(c.Request.Context(), userID, req)
	if err != nil {
		h.handleUserError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Profile updated successfully", resp)
}

// ChangePassword godoc
//
//	@Summary		Change the current user's password
//	@Description	Requires the current password. On success, every existing session (refresh token) is revoked, including the one used to make this request.
//	@Tags			users
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.ChangePasswordRequest	true	"Current and new password"
//	@Success		200		{object}	utils.SuccessResponse
//	@Failure		400		{object}	utils.ErrorResponse	"validation failed or current password incorrect"
//	@Failure		401		{object}	utils.ErrorResponse
//	@Router			/users/me/password [put]
func (h *UserHandler) ChangePassword(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Validation failed", validator.FormatValidationErrors(err))
		return
	}

	if err := h.userService.ChangePassword(c.Request.Context(), userID, req); err != nil {
		h.handleUserError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Password changed successfully. Please log in again.", nil)
}

// DeleteAccount godoc
//
//	@Summary		Delete the current user's account
//	@Description	Soft-deletes the account and revokes every session. This cannot be undone through the API.
//	@Tags			users
//	@Security		BearerAuth
//	@Produce		json
//	@Success		200	{object}	utils.SuccessResponse
//	@Failure		401	{object}	utils.ErrorResponse
//	@Router			/users/me [delete]
func (h *UserHandler) DeleteAccount(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	if err := h.userService.DeleteAccount(c.Request.Context(), userID); err != nil {
		h.handleUserError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Account deleted successfully", nil)
}

func (h *UserHandler) handleUserError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, entity.ErrNotFound):
		utils.Error(c, http.StatusNotFound, "User not found", nil)
	case errors.Is(err, service.ErrIncorrectPassword):
		utils.Error(c, http.StatusBadRequest, "Current password is incorrect", nil)
	default:
		slog.Error("user handler: unhandled error", "error", err)
		utils.Error(c, http.StatusInternalServerError, "Something went wrong", nil)
	}
}
