// Package handler is the HTTP boundary: it decodes requests into
// DTOs, calls exactly one service method, and encodes the result
// back to JSON. A handler never touches *gorm.DB and never contains
// a business rule — if you're tempted to write an `if` here that
// isn't about HTTP concerns (status codes, binding), it belongs in
// the service layer instead.
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
)

// AuthHandler wires the AuthService into Gin. Constructed once in
// main.go and handed to the router — no package-level state, so
// nothing here survives or leaks between requests except what's on
// the injected AuthService.
type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Register godoc
//
//	@Summary		Register a new account
//	@Description	Creates a customer account and returns an access/refresh token pair. New accounts are always created with the "customer" role.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.RegisterRequest	true	"Registration payload"
//	@Success		201		{object}	utils.SuccessResponse{data=dto.AuthResponse}
//	@Failure		400		{object}	utils.ErrorResponse	"validation failed"
//	@Failure		409		{object}	utils.ErrorResponse	"email already registered"
//	@Router			/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Validation failed", validator.FormatValidationErrors(err))
		return
	}

	resp, err := h.authService.Register(c.Request.Context(), req)
	if err != nil {
		h.handleAuthError(c, err)
		return
	}

	utils.Success(c, http.StatusCreated, "Account created successfully", resp)
}

// Login godoc
//
//	@Summary		Log in
//	@Description	Exchanges an email/password pair for an access/refresh token pair.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.LoginRequest	true	"Login payload"
//	@Success		200		{object}	utils.SuccessResponse{data=dto.AuthResponse}
//	@Failure		400		{object}	utils.ErrorResponse	"validation failed"
//	@Failure		401		{object}	utils.ErrorResponse	"invalid credentials"
//	@Failure		403		{object}	utils.ErrorResponse	"account blocked"
//	@Router			/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Validation failed", validator.FormatValidationErrors(err))
		return
	}

	resp, err := h.authService.Login(c.Request.Context(), req)
	if err != nil {
		h.handleAuthError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Login successful", resp)
}

// RefreshToken godoc
//
//	@Summary		Refresh an access token
//	@Description	Exchanges a valid refresh token for a brand new access/refresh pair. The refresh token used in the request is revoked immediately (rotation) — it cannot be reused.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.RefreshTokenRequest	true	"Refresh token payload"
//	@Success		200		{object}	utils.SuccessResponse{data=dto.AuthResponse}
//	@Failure		400		{object}	utils.ErrorResponse	"validation failed"
//	@Failure		401		{object}	utils.ErrorResponse	"invalid or expired refresh token"
//	@Router			/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Validation failed", validator.FormatValidationErrors(err))
		return
	}

	resp, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		h.handleAuthError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Token refreshed successfully", resp)
}

// Logout godoc
//
//	@Summary		Log out
//	@Description	Revokes the given refresh token so it can no longer be used to obtain new access tokens. Idempotent — logging out an already-invalid token still returns success.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.LogoutRequest	true	"Logout payload"
//	@Success		200		{object}	utils.SuccessResponse
//	@Failure		400		{object}	utils.ErrorResponse	"validation failed"
//	@Router			/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var req dto.LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Validation failed", validator.FormatValidationErrors(err))
		return
	}

	if err := h.authService.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		h.handleAuthError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Logged out successfully", nil)
}

// handleAuthError maps the sentinel errors the service layer can
// return to HTTP statuses. Centralizing this in one method (instead
// of repeating the same errors.Is chain in every handler above)
// means every auth endpoint stays consistent by construction, and
// adding a new sentinel error only requires one new case, here.
func (h *AuthHandler) handleAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, entity.ErrConflict):
		utils.Error(c, http.StatusConflict, "Email is already registered", nil)
	case errors.Is(err, service.ErrInvalidCredentials):
		utils.Error(c, http.StatusUnauthorized, "Invalid email or password", nil)
	case errors.Is(err, service.ErrAccountBlocked):
		utils.Error(c, http.StatusForbidden, "This account has been blocked", nil)
	case errors.Is(err, service.ErrInvalidRefreshToken):
		utils.Error(c, http.StatusUnauthorized, "Invalid or expired refresh token", nil)
	default:
		// Anything else is unexpected (a DB timeout, a bug) — log the
		// real error for us, but never leak internal detail to the
		// client.
		slog.Error("auth handler: unhandled error", "error", err)
		utils.Error(c, http.StatusInternalServerError, "Something went wrong", nil)
	}
}
