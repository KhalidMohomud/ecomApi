package service_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/KhalidMohomud/ecomApi/internal/domain/entity"
	"github.com/KhalidMohomud/ecomApi/internal/dto"
	"github.com/KhalidMohomud/ecomApi/internal/service"
	"github.com/KhalidMohomud/ecomApi/internal/utils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Unlike internal/domain/repository's tests, these are true unit
// tests: fakeUserRepository and fakeRefreshTokenRepository below are
// plain in-memory maps that satisfy the same repository.UserRepository
// / repository.RefreshTokenRepository interfaces the real GORM
// implementations satisfy. AuthService only ever depends on those
// interfaces, so it cannot tell the difference — which means these
// tests run in milliseconds with no network, no Neon, nothing to
// spin up. We didn't reach for a mocking library (e.g.
// testify/mock) because hand-writing ~20 lines of map-backed fakes
// is simpler to read than a library's expectation-setting DSL for
// an interface this small; mocking libraries earn their keep on
// much larger interfaces.

type fakeUserRepository struct {
	byEmail map[string]*entity.User
	byID    map[uuid.UUID]*entity.User
}

func newFakeUserRepository() *fakeUserRepository {
	return &fakeUserRepository{
		byEmail: make(map[string]*entity.User),
		byID:    make(map[uuid.UUID]*entity.User),
	}
}

func (f *fakeUserRepository) Create(_ context.Context, user *entity.User) error {
	if _, exists := f.byEmail[user.Email]; exists {
		return fmt.Errorf("create user: %w", entity.ErrConflict)
	}
	user.ID = uuid.New()
	user.CreatedAt = time.Now()
	f.byEmail[user.Email] = user
	f.byID[user.ID] = user
	return nil
}

func (f *fakeUserRepository) GetByID(_ context.Context, id uuid.UUID) (*entity.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return nil, fmt.Errorf("get user by id: %w", entity.ErrNotFound)
	}
	return u, nil
}

func (f *fakeUserRepository) GetByEmail(_ context.Context, email string) (*entity.User, error) {
	u, ok := f.byEmail[email]
	if !ok {
		return nil, fmt.Errorf("get user by email: %w", entity.ErrNotFound)
	}
	return u, nil
}

func (f *fakeUserRepository) Update(_ context.Context, user *entity.User) error {
	f.byEmail[user.Email] = user
	f.byID[user.ID] = user
	return nil
}

func (f *fakeUserRepository) Delete(_ context.Context, id uuid.UUID) error {
	u, ok := f.byID[id]
	if !ok {
		return fmt.Errorf("delete user: %w", entity.ErrNotFound)
	}
	delete(f.byID, id)
	delete(f.byEmail, u.Email)
	return nil
}

func (f *fakeUserRepository) List(_ context.Context, _, _ int) ([]entity.User, int64, error) {
	return nil, 0, nil
}

type fakeRefreshTokenRepository struct {
	byID   map[uuid.UUID]*entity.RefreshToken
	byHash map[string]*entity.RefreshToken
}

func newFakeRefreshTokenRepository() *fakeRefreshTokenRepository {
	return &fakeRefreshTokenRepository{
		byID:   make(map[uuid.UUID]*entity.RefreshToken),
		byHash: make(map[string]*entity.RefreshToken),
	}
}

func (f *fakeRefreshTokenRepository) Create(_ context.Context, token *entity.RefreshToken) error {
	token.ID = uuid.New()
	f.byID[token.ID] = token
	f.byHash[token.TokenHash] = token
	return nil
}

func (f *fakeRefreshTokenRepository) GetValidByHash(_ context.Context, hash string) (*entity.RefreshToken, error) {
	t, ok := f.byHash[hash]
	if !ok || !t.IsValid() {
		return nil, fmt.Errorf("get refresh token: %w", entity.ErrNotFound)
	}
	return t, nil
}

func (f *fakeRefreshTokenRepository) Revoke(_ context.Context, id uuid.UUID) error {
	t, ok := f.byID[id]
	if !ok {
		return fmt.Errorf("revoke refresh token: %w", entity.ErrNotFound)
	}
	now := time.Now()
	t.RevokedAt = &now
	return nil
}

func (f *fakeRefreshTokenRepository) RevokeAllForUser(_ context.Context, userID uuid.UUID) error {
	now := time.Now()
	for _, t := range f.byID {
		if t.UserID == userID && t.RevokedAt == nil {
			t.RevokedAt = &now
		}
	}
	return nil
}

// testDeps bundles the fakes together so tests can inspect them
// after calling into the service (e.g. to grab a raw refresh token
// hash and confirm it was actually revoked).
type testDeps struct {
	users         *fakeUserRepository
	refreshTokens *fakeRefreshTokenRepository
	svc           service.AuthService
}

func newTestDeps() testDeps {
	users := newFakeUserRepository()
	refreshTokens := newFakeRefreshTokenRepository()
	tokenManager := utils.NewTokenManager("test-secret-at-least-32-characters-long", 15*time.Minute)

	return testDeps{
		users:         users,
		refreshTokens: refreshTokens,
		svc:           service.NewAuthService(users, refreshTokens, tokenManager, 30*24*time.Hour),
	}
}

func TestAuthService_Register_Success(t *testing.T) {
	deps := newTestDeps()

	resp, err := deps.svc.Register(context.Background(), dto.RegisterRequest{
		Email:     "jane@example.com",
		Password:  "correct-horse-battery-staple",
		FirstName: "Jane",
		LastName:  "Doe",
	})

	require.NoError(t, err)
	require.NotEmpty(t, resp.AccessToken)
	require.NotEmpty(t, resp.RefreshToken)
	require.Equal(t, entity.RoleCustomer, resp.User.Role, "every self-registered account must default to customer")

	stored := deps.users.byEmail["jane@example.com"]
	require.NotEqual(t, "correct-horse-battery-staple", stored.PasswordHash, "password must never be stored in plaintext")
}

func TestAuthService_Register_DuplicateEmail(t *testing.T) {
	deps := newTestDeps()
	ctx := context.Background()
	req := dto.RegisterRequest{Email: "jane@example.com", Password: "password123", FirstName: "Jane", LastName: "Doe"}

	_, err := deps.svc.Register(ctx, req)
	require.NoError(t, err)

	_, err = deps.svc.Register(ctx, req)
	require.ErrorIs(t, err, entity.ErrConflict)
}

func TestAuthService_Login_Success(t *testing.T) {
	deps := newTestDeps()
	ctx := context.Background()

	_, err := deps.svc.Register(ctx, dto.RegisterRequest{Email: "jane@example.com", Password: "password123", FirstName: "Jane", LastName: "Doe"})
	require.NoError(t, err)

	resp, err := deps.svc.Login(ctx, dto.LoginRequest{Email: "jane@example.com", Password: "password123"})
	require.NoError(t, err)
	require.NotEmpty(t, resp.AccessToken)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	deps := newTestDeps()
	ctx := context.Background()
	_, err := deps.svc.Register(ctx, dto.RegisterRequest{Email: "jane@example.com", Password: "password123", FirstName: "Jane", LastName: "Doe"})
	require.NoError(t, err)

	_, err = deps.svc.Login(ctx, dto.LoginRequest{Email: "jane@example.com", Password: "wrong-password"})
	require.ErrorIs(t, err, service.ErrInvalidCredentials)
}

func TestAuthService_Login_UnknownEmailReturnsSameErrorAsWrongPassword(t *testing.T) {
	deps := newTestDeps()

	_, err := deps.svc.Login(context.Background(), dto.LoginRequest{Email: "nobody@example.com", Password: "whatever"})

	// Asserting this specific error (not just "an error") is the
	// point: it proves login can't be used to enumerate which emails
	// are registered.
	require.ErrorIs(t, err, service.ErrInvalidCredentials)
}

func TestAuthService_Login_BlockedAccount(t *testing.T) {
	deps := newTestDeps()
	ctx := context.Background()
	_, err := deps.svc.Register(ctx, dto.RegisterRequest{Email: "jane@example.com", Password: "password123", FirstName: "Jane", LastName: "Doe"})
	require.NoError(t, err)

	deps.users.byEmail["jane@example.com"].IsActive = false

	_, err = deps.svc.Login(ctx, dto.LoginRequest{Email: "jane@example.com", Password: "password123"})
	require.ErrorIs(t, err, service.ErrAccountBlocked)
}

func TestAuthService_RefreshToken_RotatesAndInvalidatesOldToken(t *testing.T) {
	deps := newTestDeps()
	ctx := context.Background()
	registerResp, err := deps.svc.Register(ctx, dto.RegisterRequest{Email: "jane@example.com", Password: "password123", FirstName: "Jane", LastName: "Doe"})
	require.NoError(t, err)

	refreshResp, err := deps.svc.RefreshToken(ctx, registerResp.RefreshToken)
	require.NoError(t, err)
	require.NotEqual(t, registerResp.RefreshToken, refreshResp.RefreshToken, "refresh must issue a brand new refresh token")

	// The token used to refresh must now be dead — this is what makes
	// stolen-token reuse detectable instead of silently working forever.
	_, err = deps.svc.RefreshToken(ctx, registerResp.RefreshToken)
	require.ErrorIs(t, err, service.ErrInvalidRefreshToken)

	// But the newly issued one must work.
	_, err = deps.svc.RefreshToken(ctx, refreshResp.RefreshToken)
	require.NoError(t, err)
}

func TestAuthService_RefreshToken_InvalidToken(t *testing.T) {
	deps := newTestDeps()

	_, err := deps.svc.RefreshToken(context.Background(), "not-a-real-token")

	require.ErrorIs(t, err, service.ErrInvalidRefreshToken)
}

func TestAuthService_Logout_RevokesTokenAndIsIdempotent(t *testing.T) {
	deps := newTestDeps()
	ctx := context.Background()
	registerResp, err := deps.svc.Register(ctx, dto.RegisterRequest{Email: "jane@example.com", Password: "password123", FirstName: "Jane", LastName: "Doe"})
	require.NoError(t, err)

	require.NoError(t, deps.svc.Logout(ctx, registerResp.RefreshToken))

	_, err = deps.svc.RefreshToken(ctx, registerResp.RefreshToken)
	require.ErrorIs(t, err, service.ErrInvalidRefreshToken, "a logged-out token must not be usable to refresh")

	// Logging out again with the same (already-revoked) token must
	// succeed rather than error — logout is idempotent by design.
	require.NoError(t, deps.svc.Logout(ctx, registerResp.RefreshToken))
}
