package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/KhalidMohomud/ecomApi/internal/domain/entity"
	"github.com/KhalidMohomud/ecomApi/internal/middleware"
	"github.com/KhalidMohomud/ecomApi/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// These tests never touch a database or even a real HTTP server —
// gin.New() plus httptest.NewRecorder() is enough to drive a
// middleware chain in-process and inspect the resulting response.
// That's only possible because Auth depends on *utils.TokenManager
// as a parameter rather than a package-level secret: each test
// builds its own manager with a throwaway key.

func init() {
	gin.SetMode(gin.TestMode) // silences gin's per-route startup log noise in test output
}

func newAuthTestRouter(tm *utils.TokenManager, extra ...gin.HandlerFunc) *gin.Engine {
	router := gin.New()
	handlers := append([]gin.HandlerFunc{middleware.Auth(tm)}, extra...)
	handlers = append(handlers, func(c *gin.Context) {
		userID, _ := middleware.UserIDFromContext(c)
		role, _ := middleware.RoleFromContext(c)
		c.JSON(http.StatusOK, gin.H{"user_id": userID.String(), "role": role})
	})
	router.GET("/protected", handlers...)
	return router
}

func TestAuth_MissingHeader(t *testing.T) {
	tm := utils.NewTokenManager("test-secret-at-least-32-characters-long", 15*time.Minute)
	router := newAuthTestRouter(tm)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuth_MalformedHeader(t *testing.T) {
	tm := utils.NewTokenManager("test-secret-at-least-32-characters-long", 15*time.Minute)
	router := newAuthTestRouter(tm)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "not-a-bearer-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuth_InvalidSignature(t *testing.T) {
	issuer := utils.NewTokenManager("secret-a-at-least-32-characters-long!!!", 15*time.Minute)
	verifier := utils.NewTokenManager("secret-b-completely-different-32-chars!!", 15*time.Minute)
	router := newAuthTestRouter(verifier)

	token, err := issuer.GenerateAccessToken(uuid.New(), entity.RoleCustomer)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code, "a token signed with a different secret must be rejected")
}

func TestAuth_ExpiredToken(t *testing.T) {
	// A negative TTL produces a token whose exp is already in the past.
	tm := utils.NewTokenManager("test-secret-at-least-32-characters-long", -1*time.Minute)
	router := newAuthTestRouter(tm)

	token, err := tm.GenerateAccessToken(uuid.New(), entity.RoleCustomer)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuth_ValidToken_SetsUserIDAndRoleOnContext(t *testing.T) {
	tm := utils.NewTokenManager("test-secret-at-least-32-characters-long", 15*time.Minute)
	router := newAuthTestRouter(tm)

	userID := uuid.New()
	token, err := tm.GenerateAccessToken(userID, entity.RoleAdmin)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), userID.String())
	require.Contains(t, rec.Body.String(), string(entity.RoleAdmin))
}

func TestRequireRole_DeniesWrongRole(t *testing.T) {
	tm := utils.NewTokenManager("test-secret-at-least-32-characters-long", 15*time.Minute)
	router := newAuthTestRouter(tm, middleware.RequireRole(entity.RoleAdmin))

	token, err := tm.GenerateAccessToken(uuid.New(), entity.RoleCustomer)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRequireRole_AllowsMatchingRole(t *testing.T) {
	tm := utils.NewTokenManager("test-secret-at-least-32-characters-long", 15*time.Minute)
	router := newAuthTestRouter(tm, middleware.RequireRole(entity.RoleAdmin, entity.RoleModerator))

	token, err := tm.GenerateAccessToken(uuid.New(), entity.RoleModerator)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}
