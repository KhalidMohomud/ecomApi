package service

import (
	"context"
	"fmt"

	"github.com/KhalidMohomud/ecomApi/internal/domain/repository"
	"github.com/KhalidMohomud/ecomApi/internal/dto"
	"github.com/KhalidMohomud/ecomApi/internal/utils"
	"github.com/google/uuid"
)

// UserService covers the self-service account operations a logged-in
// user performs on their own account. (Admin operations on *other*
// users — list, block, delete — are a separate service, built in a
// later step, even though both end up calling UserRepository; who is
// allowed to call each one is a completely different question, and
// keeping them apart means the admin service can carry its own
// audit-logging/authorization concerns without complicating this one.)
type UserService interface {
	GetProfile(ctx context.Context, userID uuid.UUID) (*dto.UserResponse, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, req dto.UpdateProfileRequest) (*dto.UserResponse, error)
	ChangePassword(ctx context.Context, userID uuid.UUID, req dto.ChangePasswordRequest) error
	DeleteAccount(ctx context.Context, userID uuid.UUID) error
}

type userService struct {
	userRepo         repository.UserRepository
	refreshTokenRepo repository.RefreshTokenRepository
}

func NewUserService(userRepo repository.UserRepository, refreshTokenRepo repository.RefreshTokenRepository) UserService {
	return &userService{userRepo: userRepo, refreshTokenRepo: refreshTokenRepo}
}

func (s *userService) GetProfile(ctx context.Context, userID uuid.UUID) (*dto.UserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}

	resp := dto.NewUserResponse(user)
	return &resp, nil
}

func (s *userService) UpdateProfile(ctx context.Context, userID uuid.UUID, req dto.UpdateProfileRequest) (*dto.UserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}

	// Load-modify-save on the fetched entity, touching only the
	// fields UpdateProfileRequest exposes. Role, Email, IsActive, etc.
	// are left exactly as they were on the row we just read.
	user.FirstName = req.FirstName
	user.LastName = req.LastName
	user.Phone = req.Phone

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}

	resp := dto.NewUserResponse(user)
	return &resp, nil
}

func (s *userService) ChangePassword(ctx context.Context, userID uuid.UUID, req dto.ChangePasswordRequest) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("change password: %w", err)
	}

	if !utils.CheckPassword(user.PasswordHash, req.CurrentPassword) {
		return ErrIncorrectPassword
	}

	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("change password: %w", err)
	}
	user.PasswordHash = hashedPassword

	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("change password: %w", err)
	}

	// A leaked-password scenario is exactly what changing your
	// password is meant to fix — so every existing session gets
	// signed out, including the one that just made this request. The
	// client is expected to already hold the new credentials and log
	// in again to get a fresh token pair.
	if err := s.refreshTokenRepo.RevokeAllForUser(ctx, userID); err != nil {
		return fmt.Errorf("change password: revoking sessions: %w", err)
	}

	return nil
}

func (s *userService) DeleteAccount(ctx context.Context, userID uuid.UUID) error {
	if err := s.userRepo.Delete(ctx, userID); err != nil {
		return fmt.Errorf("delete account: %w", err)
	}

	if err := s.refreshTokenRepo.RevokeAllForUser(ctx, userID); err != nil {
		return fmt.Errorf("delete account: revoking sessions: %w", err)
	}

	return nil
}
