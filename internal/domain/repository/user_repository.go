// Package repository contains the concrete data-access layer: one
// type per entity, each responsible only for translating between
// Go values and SQL rows. No business rules live here — a
// repository doesn't decide whether an email is allowed to
// register, it just stores and retrieves rows. That decision
// belongs to internal/service.
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/KhalidMohomud/ecomApi/internal/domain/entity"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRepository defines every persistence operation available for
// User entities.
//
// The service layer will depend on this interface, never on the
// concrete userRepository struct below or on *gorm.DB directly.
// That indirection is what makes the service layer testable: a test
// can hand it an in-memory fake that implements this same interface
// instead of talking to real Postgres.
type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
	Update(ctx context.Context, user *entity.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, offset, limit int) ([]entity.User, int64, error)
	// Stats returns aggregate counts for the admin dashboard.
	Stats(ctx context.Context) (*entity.UserStats, error)
}

// userRepository is the GORM/PostgreSQL implementation of
// UserRepository. It is unexported — code outside this package
// can't reference the struct type at all, only the interface,
// which is exactly the point.
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository is the constructor function for userRepository.
// It returns the interface type, not *userRepository, so callers
// are structurally prevented from depending on anything beyond the
// interface's method set.
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *entity.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return fmt.Errorf("create user: %w", entity.ErrConflict)
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	var user entity.User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("get user by id: %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	var user entity.User
	if err := r.db.WithContext(ctx).First(&user, "email = ?", email).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("get user by email: %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return &user, nil
}

// Update overwrites every column of an existing user with the
// values currently on the struct.
//
// It deliberately uses Save, not Updates. GORM's Updates(struct)
// skips any field holding its Go zero value (false, "", 0, ...) —
// so Updates(&user) right after `user.IsActive = false` would
// silently NOT write is_active, because false is bool's zero value.
// Save has no such exception: it always writes every column. Since
// this method's contract is "persist the full state of this
// struct", Save is the correct choice; Updates is for callers that
// intentionally want a partial patch.
func (r *userRepository) Update(ctx context.Context, user *entity.User) error {
	if err := r.db.WithContext(ctx).Save(user).Error; err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

// Delete soft-deletes a user: because entity.User embeds
// gorm.DeletedAt, this issues an UPDATE that sets deleted_at rather
// than a DELETE that removes the row.
func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&entity.User{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete user: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("delete user: %w", entity.ErrNotFound)
	}
	return nil
}

// List returns a page of users ordered newest-first, along with the
// total row count so the caller (eventually the admin "list users"
// handler) can compute total pages.
func (r *userRepository) List(ctx context.Context, offset, limit int) ([]entity.User, int64, error) {
	var users []entity.User
	var total int64

	if err := r.db.WithContext(ctx).Model(&entity.User{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	if err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}

	return users, total, nil
}

// Stats computes user counts in a single query using Postgres's
// FILTER clause (COUNT(*) FILTER (WHERE ...)) rather than three
// separate round trips. GORM's soft-delete scope still applies here
// exactly as it does to Find/First — Model(&entity.User{}) is what
// tells GORM which model's callbacks (including the deleted_at IS
// NULL filter) to attach to this query.
func (r *userRepository) Stats(ctx context.Context) (*entity.UserStats, error) {
	var stats entity.UserStats
	err := r.db.WithContext(ctx).
		Model(&entity.User{}).
		Select("COUNT(*) AS total, COUNT(*) FILTER (WHERE is_active) AS active, COUNT(*) FILTER (WHERE NOT is_active) AS blocked").
		Scan(&stats).Error
	if err != nil {
		return nil, fmt.Errorf("user stats: %w", err)
	}
	return &stats, nil
}
