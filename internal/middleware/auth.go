package middleware

import (
	"net/http"
	"strings"

	"github.com/KhalidMohomud/ecomApi/internal/domain/entity"
	"github.com/KhalidMohomud/ecomApi/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Keys used to stash values on gin.Context inside Auth, and read
// back by handlers/other middleware via the helper functions below.
// Typed constants instead of bare string literals scattered across
// files — a typo in one place would silently fail to find the value
// with no compiler error.
const (
	contextKeyUserID = "auth.userID"
	contextKeyRole   = "auth.role"
)

// Auth returns middleware that requires a valid JWT access token on
// the request and, if present, makes the authenticated user's ID and
// role available to downstream handlers via UserIDFromContext /
// RoleFromContext.
//
// It takes a *utils.TokenManager as a parameter rather than reaching
// for a global — same dependency-injection pattern as everywhere
// else in this project — which is also what makes this middleware
// unit-testable with a throwaway secret instead of the app's real one.
func Auth(tokenManager *utils.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			utils.Error(c, http.StatusUnauthorized, "Authorization header is required", nil)
			c.Abort() // stop the chain — none of the actual route handlers run
			return
		}

		const prefix = "Bearer "
		if !strings.HasPrefix(header, prefix) {
			utils.Error(c, http.StatusUnauthorized, `Authorization header must be in the form "Bearer <token>"`, nil)
			c.Abort()
			return
		}
		rawToken := strings.TrimPrefix(header, prefix)

		claims, err := tokenManager.ParseAccessToken(rawToken)
		if err != nil {
			utils.Error(c, http.StatusUnauthorized, "Invalid or expired access token", nil)
			c.Abort()
			return
		}

		userID, err := uuid.Parse(claims.Subject)
		if err != nil {
			// Should be unreachable — we always mint the Subject
			// claim from a uuid.UUID in GenerateAccessToken. Treated
			// as an auth failure, not a 500, since a client cannot
			// hit this except by crafting a malformed token.
			utils.Error(c, http.StatusUnauthorized, "Invalid access token", nil)
			c.Abort()
			return
		}

		c.Set(contextKeyUserID, userID)
		c.Set(contextKeyRole, claims.Role)
		c.Next()
	}
}

// RequireRole returns middleware that only allows the request
// through if the authenticated user's role is one of allowed. It
// must run after Auth — it reads the role Auth stored on the
// context, and denies the request if that's missing.
//
// This is what turns "logged in" into "logged in AND allowed to do
// this" — used starting with the Admin endpoints in the next step,
// e.g. RequireRole(entity.RoleAdmin).
func RequireRole(allowed ...entity.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, ok := RoleFromContext(c)
		if !ok {
			utils.Error(c, http.StatusForbidden, "Access denied", nil)
			c.Abort()
			return
		}

		for _, r := range allowed {
			if role == r {
				c.Next()
				return
			}
		}

		utils.Error(c, http.StatusForbidden, "You do not have permission to perform this action", nil)
		c.Abort()
	}
}

// UserIDFromContext retrieves the authenticated user's ID set by
// Auth. The bool return mirrors the "comma ok" idiom used throughout
// Go (map lookups, type assertions) — ok is false if Auth never ran
// on this request, which a handler should treat as a bug, not a
// silent zero-value UUID.
func UserIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	v, exists := c.Get(contextKeyUserID)
	if !exists {
		return uuid.Nil, false
	}
	id, ok := v.(uuid.UUID)
	return id, ok
}

// RoleFromContext retrieves the authenticated user's role set by Auth.
func RoleFromContext(c *gin.Context) (entity.Role, bool) {
	v, exists := c.Get(contextKeyRole)
	if !exists {
		return "", false
	}
	role, ok := v.(entity.Role)
	return role, ok
}
