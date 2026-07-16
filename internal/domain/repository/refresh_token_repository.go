package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/KhalidMohomud/ecomApi/internal/domain/entity"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RefreshTokenRepository persists the server-side half of the
// refresh-token flow. See entity.RefreshToken and
// internal/utils/token.go for why refresh tokens are stored at all,
// unlike access tokens.
type RefreshTokenRepository interface {
	Create(ctx context.Context, token *entity.RefreshToken) error
	// GetValidByHash returns the token row matching hash, but only if
	// it is not revoked and not expired. An expired-but-present row
	// and a genuinely nonexistent hash both surface as
	// entity.ErrNotFound — the caller only needs to know "this token
	// cannot be used," not why.
	GetValidByHash(ctx context.Context, hash string) (*entity.RefreshToken, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	// RevokeAllForUser invalidates every still-valid session a user
	// has. Used by change-password and delete-account: if your
	// credentials changed, every device logged in under the old
	// credentials should be signed out, not just the one that made
	// the request.
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}

type refreshTokenRepository struct {
	db *gorm.DB
}

func NewRefreshTokenRepository(db *gorm.DB) RefreshTokenRepository {
	return &refreshTokenRepository{db: db}
}

func (r *refreshTokenRepository) Create(ctx context.Context, token *entity.RefreshToken) error {
	if err := r.db.WithContext(ctx).Create(token).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return fmt.Errorf("create refresh token: %w", entity.ErrConflict)
		}
		return fmt.Errorf("create refresh token: %w", err)
	}
	return nil
}

func (r *refreshTokenRepository) GetValidByHash(ctx context.Context, hash string) (*entity.RefreshToken, error) {
	var token entity.RefreshToken
	err := r.db.WithContext(ctx).
		Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", hash, time.Now()).
		First(&token).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("get refresh token: %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("get refresh token: %w", err)
	}
	return &token, nil
}

func (r *refreshTokenRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Model(&entity.RefreshToken{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", time.Now())
	if result.Error != nil {
		return fmt.Errorf("revoke refresh token: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("revoke refresh token: %w", entity.ErrNotFound)
	}
	return nil
}

func (r *refreshTokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	// No RowsAffected check here, unlike Revoke: "this user had zero
	// active sessions to revoke" is a completely normal outcome (e.g.
	// they were only ever logged in on the device making this very
	// request), not an error condition.
	if err := r.db.WithContext(ctx).
		Model(&entity.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", time.Now()).Error; err != nil {
		return fmt.Errorf("revoke all refresh tokens for user: %w", err)
	}
	return nil
}
