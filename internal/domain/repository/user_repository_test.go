package repository_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/KhalidMohomud/ecomApi/internal/domain/entity"
	"github.com/KhalidMohomud/ecomApi/internal/domain/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// This is an integration test: it talks to the real Neon database
// configured in .env, not a mock. That's a deliberate choice for a
// repository test — the entire point of this layer is "does our SQL
// actually do what we think," and a mock can't answer that. Unit
// tests that don't need a real database (the service layer, once it
// exists) will use a fake UserRepository instead.
//
// Package is `repository_test`, not `repository` — an external test
// package. It can only see repository's exported API (UserRepository,
// NewUserRepository), the same as any other consumer. That's a
// useful discipline: if a test needs something unexported to pass,
// that's a sign the public API is missing something, not a reason to
// reach around it.

// newTestRepo wraps the shared testDB helper (testdb_test.go) with
// the one line specific to this file: constructing a UserRepository
// from that connection.
func newTestRepo(t *testing.T) repository.UserRepository {
	t.Helper()
	return repository.NewUserRepository(testDB(t))
}

// uniqueEmail avoids collisions between test runs (and between
// parallel tests) without needing to truncate the table.
func uniqueEmail() string {
	return fmt.Sprintf("test-%s@example.com", uuid.NewString())
}

// repoRootEnvFile locates the module's .env file regardless of
// which package directory `go test` happens to be running from.
//
// This matters because `go test` always sets the working directory
// to the package under test (here, internal/domain/repository), not
// the repository root — a bare config.Load(".env") would look for
// internal/domain/repository/.env, which doesn't exist. We instead
// walk upward from the current directory until we find go.mod,
// which by definition marks the module root, and load .env from
// there. Every future *_test.go file that needs .env can reuse this
// same pattern.
func repoRootEnvFile(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	require.NoError(t, err)

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Join(dir, ".env")
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, dir, parent, "walked up to filesystem root without finding go.mod")
		dir = parent
	}
}

func TestUserRepository_CreateAndGet(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	user := &entity.User{
		Email:        uniqueEmail(),
		PasswordHash: "hashed-password",
		FirstName:    "Ada",
		LastName:     "Lovelace",
		Role:         entity.RoleCustomer,
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, user.ID, "Postgres should have generated an ID via gen_random_uuid()")

	t.Cleanup(func() { _ = repo.Delete(ctx, user.ID) })

	byID, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, user.Email, byID.Email)
	require.Equal(t, entity.RoleCustomer, byID.Role)

	byEmail, err := repo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	require.Equal(t, user.ID, byEmail.ID)
}

func TestUserRepository_DuplicateEmailReturnsConflict(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()
	email := uniqueEmail()

	first := &entity.User{Email: email, PasswordHash: "x", FirstName: "A", LastName: "B", Role: entity.RoleCustomer}
	require.NoError(t, repo.Create(ctx, first))
	t.Cleanup(func() { _ = repo.Delete(ctx, first.ID) })

	second := &entity.User{Email: email, PasswordHash: "x", FirstName: "C", LastName: "D", Role: entity.RoleCustomer}
	err := repo.Create(ctx, second)

	require.Error(t, err)
	require.ErrorIs(t, err, entity.ErrConflict)
}

func TestUserRepository_GetByID_NotFound(t *testing.T) {
	repo := newTestRepo(t)

	_, err := repo.GetByID(context.Background(), uuid.New())

	require.Error(t, err)
	require.ErrorIs(t, err, entity.ErrNotFound)
}

func TestUserRepository_UpdateWritesZeroValueFields(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	user := &entity.User{
		Email:        uniqueEmail(),
		PasswordHash: "x",
		FirstName:    "Grace",
		LastName:     "Hopper",
		Role:         entity.RoleCustomer,
		IsActive:     true,
	}
	require.NoError(t, repo.Create(ctx, user))
	t.Cleanup(func() { _ = repo.Delete(ctx, user.ID) })

	// IsActive:false is bool's zero value. This specifically exercises
	// the Save-vs-Updates distinction documented on Update: if the
	// repository ever switched to db.Updates(user), this assertion
	// would fail because GORM would silently skip writing `false`.
	user.FirstName = "Updated"
	user.IsActive = false
	require.NoError(t, repo.Update(ctx, user))

	fetched, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, "Updated", fetched.FirstName)
	require.False(t, fetched.IsActive, "zero-value field (IsActive=false) must still be persisted")
}

func TestUserRepository_DeleteIsSoftAndExcludesFromReads(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	user := &entity.User{Email: uniqueEmail(), PasswordHash: "x", FirstName: "Margaret", LastName: "Hamilton", Role: entity.RoleCustomer}
	require.NoError(t, repo.Create(ctx, user))

	require.NoError(t, repo.Delete(ctx, user.ID))

	_, err := repo.GetByID(ctx, user.ID)
	require.ErrorIs(t, err, entity.ErrNotFound, "soft-deleted users must be excluded from normal reads")

	// Deleting again should report not-found rather than succeeding
	// silently, since RowsAffected will be 0 the second time.
	err = repo.Delete(ctx, user.ID)
	require.ErrorIs(t, err, entity.ErrNotFound)
}

func TestUserRepository_List(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	var created []uuid.UUID
	for i := 0; i < 3; i++ {
		user := &entity.User{Email: uniqueEmail(), PasswordHash: "x", FirstName: "List", LastName: "Test", Role: entity.RoleCustomer}
		require.NoError(t, repo.Create(ctx, user))
		created = append(created, user.ID)
	}
	t.Cleanup(func() {
		for _, id := range created {
			_ = repo.Delete(ctx, id)
		}
	})

	users, total, err := repo.List(ctx, 0, 2)
	require.NoError(t, err)
	require.Len(t, users, 2, "limit=2 should return exactly 2 rows")
	require.GreaterOrEqual(t, total, int64(3), "total should count all users, not just the current page")
}
