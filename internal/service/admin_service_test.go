package service_test

import (
	"context"
	"testing"

	"github.com/KhalidMohomud/ecomApi/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type adminServiceTestDeps struct {
	users         *fakeUserRepository
	refreshTokens *fakeRefreshTokenRepository
	svc           service.AdminService
}

func newAdminServiceTestDeps() adminServiceTestDeps {
	users := newFakeUserRepository()
	refreshTokens := newFakeRefreshTokenRepository()
	return adminServiceTestDeps{
		users:         users,
		refreshTokens: refreshTokens,
		svc:           service.NewAdminService(users, refreshTokens),
	}
}

func TestAdminService_GetDashboard(t *testing.T) {
	deps := newAdminServiceTestDeps()
	registerTestUser(t, userServiceTestDeps{users: deps.users}, "active@example.com", "password123")
	registerTestUser(t, userServiceTestDeps{users: deps.users}, "blocked@example.com", "password123")
	deps.users.byEmail["blocked@example.com"].IsActive = false

	resp, err := deps.svc.GetDashboard(context.Background())

	require.NoError(t, err)
	require.Equal(t, int64(2), resp.TotalUsers)
	require.Equal(t, int64(1), resp.ActiveUsers)
	require.Equal(t, int64(1), resp.BlockedUsers)
}

func TestAdminService_ListUsers_ComputesTotalPages(t *testing.T) {
	deps := newAdminServiceTestDeps()
	for i := 0; i < 5; i++ {
		registerTestUser(t, userServiceTestDeps{users: deps.users}, uuid.NewString()+"@example.com", "password123")
	}

	resp, err := deps.svc.ListUsers(context.Background(), 1, 2)

	require.NoError(t, err)
	require.Len(t, resp.Items, 2, "page size 2 should return exactly 2 items")
	require.Equal(t, int64(5), resp.Total)
	require.Equal(t, 3, resp.TotalPages, "5 items at page size 2 must round up to 3 pages")
}

func TestAdminService_BlockUser_CannotBlockSelf(t *testing.T) {
	deps := newAdminServiceTestDeps()
	registerTestUser(t, userServiceTestDeps{users: deps.users}, "admin@example.com", "password123")
	admin := deps.users.byEmail["admin@example.com"]

	err := deps.svc.BlockUser(context.Background(), admin.ID, admin.ID)

	require.ErrorIs(t, err, service.ErrCannotModifySelf)
}

func TestAdminService_BlockUser_DeactivatesAndRevokesSessions(t *testing.T) {
	deps := newAdminServiceTestDeps()
	registerTestUser(t, userServiceTestDeps{users: deps.users}, "admin@example.com", "password123")
	registerTestUser(t, userServiceTestDeps{users: deps.users}, "target@example.com", "password123")
	admin := deps.users.byEmail["admin@example.com"]
	target := deps.users.byEmail["target@example.com"]

	hash := createTestSession(t, deps.refreshTokens, target.ID)

	err := deps.svc.BlockUser(context.Background(), admin.ID, target.ID)

	require.NoError(t, err)
	require.False(t, deps.users.byID[target.ID].IsActive)
	_, err = deps.refreshTokens.GetValidByHash(context.Background(), hash)
	require.Error(t, err, "blocking a user must revoke their existing sessions")
}

func TestAdminService_UnblockUser_Reactivates(t *testing.T) {
	deps := newAdminServiceTestDeps()
	registerTestUser(t, userServiceTestDeps{users: deps.users}, "admin@example.com", "password123")
	registerTestUser(t, userServiceTestDeps{users: deps.users}, "target@example.com", "password123")
	admin := deps.users.byEmail["admin@example.com"]
	target := deps.users.byEmail["target@example.com"]
	deps.users.byID[target.ID].IsActive = false

	err := deps.svc.UnblockUser(context.Background(), admin.ID, target.ID)

	require.NoError(t, err)
	require.True(t, deps.users.byID[target.ID].IsActive)
}

func TestAdminService_DeleteUser_CannotDeleteSelf(t *testing.T) {
	deps := newAdminServiceTestDeps()
	registerTestUser(t, userServiceTestDeps{users: deps.users}, "admin@example.com", "password123")
	admin := deps.users.byEmail["admin@example.com"]

	err := deps.svc.DeleteUser(context.Background(), admin.ID, admin.ID)

	require.ErrorIs(t, err, service.ErrCannotModifySelf)
}

func TestAdminService_DeleteUser_RemovesAndRevokesSessions(t *testing.T) {
	deps := newAdminServiceTestDeps()
	registerTestUser(t, userServiceTestDeps{users: deps.users}, "admin@example.com", "password123")
	registerTestUser(t, userServiceTestDeps{users: deps.users}, "target@example.com", "password123")
	admin := deps.users.byEmail["admin@example.com"]
	target := deps.users.byEmail["target@example.com"]
	hash := createTestSession(t, deps.refreshTokens, target.ID)

	err := deps.svc.DeleteUser(context.Background(), admin.ID, target.ID)

	require.NoError(t, err)
	_, err = deps.users.GetByID(context.Background(), target.ID)
	require.Error(t, err, "deleted user must no longer be fetchable")
	_, err = deps.refreshTokens.GetValidByHash(context.Background(), hash)
	require.Error(t, err, "deleting a user must revoke their existing sessions")
}
