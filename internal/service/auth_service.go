// Package service contains all business logic. This is the only
// layer allowed to make decisions ("is this email taken", "is this
// password correct", "should this token be rotated") — repositories
// only fetch and store, handlers only translate HTTP <-> Go values.
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/KhalidMohomud/ecomApi/internal/domain/entity"
	"github.com/KhalidMohomud/ecomApi/internal/domain/repository"
	"github.com/KhalidMohomud/ecomApi/internal/dto"
	"github.com/KhalidMohomud/ecomApi/internal/utils"
)

// AuthService defines the auth use cases. Handlers depend on this
// interface, not on *authService — exactly the same reasoning as
// repository.UserRepository: it's what lets us unit-test handlers
// (and this service itself, against fake repositories) without a
// real database or real bcrypt/JWT work.
type AuthService interface {
	Register(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error)
	Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error)
	RefreshToken(ctx context.Context, rawRefreshToken string) (*dto.AuthResponse, error)
	Logout(ctx context.Context, rawRefreshToken string) error
}

type authService struct {
	userRepo         repository.UserRepository
	refreshTokenRepo repository.RefreshTokenRepository
	tokenManager     *utils.TokenManager
	refreshTokenTTL  time.Duration
}

func NewAuthService(
	userRepo repository.UserRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	tokenManager *utils.TokenManager,
	refreshTokenTTL time.Duration,
) AuthService {
	return &authService{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		tokenManager:     tokenManager,
		refreshTokenTTL:  refreshTokenTTL,
	}
}

func (s *authService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error) {
	_, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err == nil {
		return nil, fmt.Errorf("register: %w", entity.ErrConflict)
	}
	if !errors.Is(err, entity.ErrNotFound) {
		return nil, fmt.Errorf("register: checking existing email: %w", err)
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("register: %w", err)
	}

	// Explicit field-by-field construction, not a struct conversion
	// from req. This is the mass-assignment guard: RegisterRequest
	// has no Role or IsActive field to begin with, so there is
	// nothing a client could set beyond these four values, and every
	// new account is unconditionally a customer.
	user := &entity.User{
		Email:        req.Email,
		PasswordHash: hashedPassword,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Role:         entity.RoleCustomer,
		// Set explicitly rather than relying on the `default:true`
		// GORM tag on entity.User.IsActive to fill this in — that
		// mechanism only fires because it's a zero-value omission
		// trick specific to GORM's INSERT, which is easy to forget
		// and doesn't apply if the repository is ever swapped for a
		// fake in a test or a different persistence backend. A new
		// account being active is a business rule, so it belongs
		// here, explicit and visible, not implied by an ORM tag.
		IsActive: true,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("register: %w", err)
	}

	return s.issueTokens(ctx, user)
}

func (s *authService) Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error) {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("login: %w", err)
	}

	if !utils.CheckPassword(user.PasswordHash, req.Password) {
		return nil, ErrInvalidCredentials
	}

	if user.IsBlocked() {
		return nil, ErrAccountBlocked
	}

	return s.issueTokens(ctx, user)
}

func (s *authService) RefreshToken(ctx context.Context, rawRefreshToken string) (*dto.AuthResponse, error) {
	hash := utils.HashToken(rawRefreshToken)

	stored, err := s.refreshTokenRepo.GetValidByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return nil, ErrInvalidRefreshToken
		}
		return nil, fmt.Errorf("refresh token: %w", err)
	}

	user, err := s.userRepo.GetByID(ctx, stored.UserID)
	if err != nil {
		return nil, fmt.Errorf("refresh token: %w", err)
	}
	if user.IsBlocked() {
		return nil, ErrAccountBlocked
	}

	// Rotation: the token that was just used is revoked immediately,
	// and a brand new pair is issued. If someone steals a refresh
	// token and uses it, the legitimate owner's next refresh attempt
	// will fail (the token they had is now revoked) — a visible signal
	// something is wrong, instead of both parties silently sharing a
	// long-lived credential.
	if err := s.refreshTokenRepo.Revoke(ctx, stored.ID); err != nil {
		return nil, fmt.Errorf("refresh token: %w", err)
	}

	return s.issueTokens(ctx, user)
}

func (s *authService) Logout(ctx context.Context, rawRefreshToken string) error {
	hash := utils.HashToken(rawRefreshToken)

	stored, err := s.refreshTokenRepo.GetValidByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			// Already invalid or already logged out — logout is
			// idempotent, so this is success, not an error.
			return nil
		}
		return fmt.Errorf("logout: %w", err)
	}

	if err := s.refreshTokenRepo.Revoke(ctx, stored.ID); err != nil {
		return fmt.Errorf("logout: %w", err)
	}
	return nil
}

// issueTokens generates a fresh access/refresh pair for user and
// persists the refresh token. It's the one place both Register and
// Login (and RefreshToken, after rotation) converge, so the token
// issuance logic exists exactly once.
func (s *authService) issueTokens(ctx context.Context, user *entity.User) (*dto.AuthResponse, error) {
	accessToken, err := s.tokenManager.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		return nil, fmt.Errorf("issue tokens: %w", err)
	}

	rawRefreshToken, hash, err := utils.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("issue tokens: %w", err)
	}

	refreshToken := &entity.RefreshToken{
		UserID:    user.ID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(s.refreshTokenTTL),
	}
	if err := s.refreshTokenRepo.Create(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("issue tokens: %w", err)
	}

	return &dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		ExpiresIn:    int64(s.tokenManager.AccessTTL().Seconds()),
		User:         dto.NewUserResponse(user),
	}, nil
}
