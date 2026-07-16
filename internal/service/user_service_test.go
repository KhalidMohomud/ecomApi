package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/KhalidMohomud/ecomApi/internal/domain/entity"
	"github.com/KhalidMohomud/ecomApi/internal/dto"
	"github.com/KhalidMohomud/ecomApi/internal/service"
	"github.com/KhalidMohomud/ecomApi/internal/utils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// createTestSession inserts a valid refresh token for userID and
// returns its hash — the same value GetValidByHash expects — so a
// test can later assert the lookup fails once the session is revoked.
func createTestSession(t *testing.T, repo *fakeRefreshTokenRepository, userID uuid.UUID) string {
	t.Helper()
	raw, hash, err := utils.GenerateRefreshToken()
	require.NoError(t, err)

	err = repo.Create(context.Background(), &entity.RefreshToken{
		UserID:    userID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	})
	require.NoError(t, err)

	return hash // GetValidByHash takes the hash, not the raw value
}

// newUserServiceTestDeps reuses fakeUserRepository and
// fakeRefreshTokenRepository defined in auth_service_test.go — same
// package, same test binary, no need to redefine them.
type userServiceTestDeps struct {
	users         *fakeUserRepository
	refreshTokens *fakeRefreshTokenRepository
	svc           service.UserService
}

func newUserServiceTestDeps() userServiceTestDeps {
	users := newFakeUserRepository()
	refreshTokens := newFakeRefreshTokenRepository()
	return userServiceTestDeps{
		users:         users,
		refreshTokens: refreshTokens,
		svc:           service.NewUserService(users, refreshTokens),
	}
}

// registerTestUser bypasses AuthService and inserts a user directly
// via the fake repository, since these tests exercise UserService in
// isolation. hashedPassword uses the real bcrypt hash so
// ChangePassword tests exercise the real verification path.
func registerTestUser(t *testing.T, deps userServiceTestDeps, email, password string) {
	t.Helper()
	hash, err := utils.HashPassword(password)
	require.NoError(t, err)

	user := &entity.User{
		Email:        email,
		PasswordHash: hash,
		FirstName:    "Jane",
		LastName:     "Doe",
		Role:         entity.RoleCustomer,
		IsActive:     true,
	}
	require.NoError(t, deps.users.Create(context.Background(), user))
}

func TestUserService_GetProfile(t *testing.T) {
	deps := newUserServiceTestDeps()
	registerTestUser(t, deps, "jane@example.com", "password123")
	user := deps.users.byEmail["jane@example.com"]

	resp, err := deps.svc.GetProfile(context.Background(), user.ID)

	require.NoError(t, err)
	require.Equal(t, "jane@example.com", resp.Email)
}

func TestUserService_UpdateProfile_OnlyTouchesAllowedFields(t *testing.T) {
	deps := newUserServiceTestDeps()
	registerTestUser(t, deps, "jane@example.com", "password123")
	user := deps.users.byEmail["jane@example.com"]
	originalRole := user.Role

	resp, err := deps.svc.UpdateProfile(context.Background(), user.ID, dto.UpdateProfileRequest{
		FirstName: "Updated",
		LastName:  "Name",
	})

	require.NoError(t, err)
	require.Equal(t, "Updated", resp.FirstName)
	require.Equal(t, originalRole, deps.users.byID[user.ID].Role, "UpdateProfile must never change Role")
	require.Equal(t, "jane@example.com", deps.users.byID[user.ID].Email, "UpdateProfile must never change Email")
}

func TestUserService_ChangePassword_WrongCurrentPassword(t *testing.T) {
	deps := newUserServiceTestDeps()
	registerTestUser(t, deps, "jane@example.com", "password123")
	user := deps.users.byEmail["jane@example.com"]

	err := deps.svc.ChangePassword(context.Background(), user.ID, dto.ChangePasswordRequest{
		CurrentPassword: "wrong-password",
		NewPassword:     "new-password123",
	})

	require.ErrorIs(t, err, service.ErrIncorrectPassword)
}

func TestUserService_ChangePassword_RevokesAllSessions(t *testing.T) {
	deps := newUserServiceTestDeps()
	registerTestUser(t, deps, "jane@example.com", "password123")
	user := deps.users.byEmail["jane@example.com"]

	// Simulate two active sessions (e.g. phone + laptop) for this user.
	hash1 := createTestSession(t, deps.refreshTokens, user.ID)
	hash2 := createTestSession(t, deps.refreshTokens, user.ID)

	err := deps.svc.ChangePassword(context.Background(), user.ID, dto.ChangePasswordRequest{
		CurrentPassword: "password123",
		NewPassword:     "new-password123",
	})
	require.NoError(t, err)

	_, err = deps.refreshTokens.GetValidByHash(context.Background(), hash1)
	require.Error(t, err, "session 1 must be revoked after password change")
	_, err = deps.refreshTokens.GetValidByHash(context.Background(), hash2)
	require.Error(t, err, "session 2 must be revoked after password change")
}

func TestUserService_DeleteAccount_SoftDeletesAndRevokesSessions(t *testing.T) {
	deps := newUserServiceTestDeps()
	registerTestUser(t, deps, "jane@example.com", "password123")
	user := deps.users.byEmail["jane@example.com"]

	hash := createTestSession(t, deps.refreshTokens, user.ID)

	err := deps.svc.DeleteAccount(context.Background(), user.ID)
	require.NoError(t, err)

	_, err = deps.svc.GetProfile(context.Background(), user.ID)
	require.Error(t, err, "deleted account must no longer be fetchable")

	_, err = deps.refreshTokens.GetValidByHash(context.Background(), hash)
	require.Error(t, err, "sessions must be revoked on account deletion")
}
