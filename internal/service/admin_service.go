package service

import (
	"context"
	"fmt"

	"github.com/KhalidMohomud/ecomApi/internal/domain/repository"
	"github.com/KhalidMohomud/ecomApi/internal/dto"
	"github.com/google/uuid"
)

// AdminService covers operations an admin performs on *other*
// users' accounts. It reuses the same UserRepository as UserService
// (Step 4) — the data being touched is identical — but stays a
// separate service because the authorization question is entirely
// different ("am I allowed to touch this account at all" vs. "am I
// touching my own account"), and because every method here takes an
// actorID for the self-action guard that UserService's methods have
// no reason to need.
type AdminService interface {
	GetDashboard(ctx context.Context) (*dto.DashboardResponse, error)
	ListUsers(ctx context.Context, page, pageSize int) (*dto.PaginatedResponse[dto.UserResponse], error)
	BlockUser(ctx context.Context, actorID, targetID uuid.UUID) error
	UnblockUser(ctx context.Context, actorID, targetID uuid.UUID) error
	DeleteUser(ctx context.Context, actorID, targetID uuid.UUID) error
}

type adminService struct {
	userRepo         repository.UserRepository
	refreshTokenRepo repository.RefreshTokenRepository
}

func NewAdminService(userRepo repository.UserRepository, refreshTokenRepo repository.RefreshTokenRepository) AdminService {
	return &adminService{userRepo: userRepo, refreshTokenRepo: refreshTokenRepo}
}

func (s *adminService) GetDashboard(ctx context.Context) (*dto.DashboardResponse, error) {
	stats, err := s.userRepo.Stats(ctx)
	if err != nil {
		return nil, fmt.Errorf("get dashboard: %w", err)
	}

	return &dto.DashboardResponse{
		TotalUsers:   stats.Total,
		ActiveUsers:  stats.Active,
		BlockedUsers: stats.Blocked,
	}, nil
}

func (s *adminService) ListUsers(ctx context.Context, page, pageSize int) (*dto.PaginatedResponse[dto.UserResponse], error) {
	offset := (page - 1) * pageSize

	users, total, err := s.userRepo.List(ctx, offset, pageSize)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	items := make([]dto.UserResponse, len(users))
	for i, u := range users {
		items[i] = dto.NewUserResponse(&u)
	}

	resp := dto.NewPaginatedResponse(items, total, page, pageSize)
	return &resp, nil
}

func (s *adminService) BlockUser(ctx context.Context, actorID, targetID uuid.UUID) error {
	if actorID == targetID {
		return ErrCannotModifySelf
	}

	user, err := s.userRepo.GetByID(ctx, targetID)
	if err != nil {
		return fmt.Errorf("block user: %w", err)
	}

	user.IsActive = false
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("block user: %w", err)
	}

	// Kick out every existing session immediately. Note this only
	// affects refresh tokens: an access token the user already holds
	// remains cryptographically valid until it expires (at most
	// AccessTokenTTL, 15 minutes by default) — that's the trade-off
	// of stateless JWTs described in internal/utils/token.go. A
	// blocked user is fully locked out the moment their access token
	// expires or they try to refresh, not necessarily instantly. If
	// instant revocation ever becomes a hard requirement, the fix is
	// a short-lived access-token blocklist checked in
	// middleware.Auth, not a redesign of this method.
	if err := s.refreshTokenRepo.RevokeAllForUser(ctx, targetID); err != nil {
		return fmt.Errorf("block user: revoking sessions: %w", err)
	}

	return nil
}

func (s *adminService) UnblockUser(ctx context.Context, actorID, targetID uuid.UUID) error {
	if actorID == targetID {
		return ErrCannotModifySelf
	}

	user, err := s.userRepo.GetByID(ctx, targetID)
	if err != nil {
		return fmt.Errorf("unblock user: %w", err)
	}

	user.IsActive = true
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("unblock user: %w", err)
	}

	return nil
}

func (s *adminService) DeleteUser(ctx context.Context, actorID, targetID uuid.UUID) error {
	if actorID == targetID {
		return ErrCannotModifySelf
	}

	if err := s.userRepo.Delete(ctx, targetID); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	if err := s.refreshTokenRepo.RevokeAllForUser(ctx, targetID); err != nil {
		return fmt.Errorf("delete user: revoking sessions: %w", err)
	}

	return nil
}
