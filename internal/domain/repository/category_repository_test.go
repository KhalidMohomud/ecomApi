package repository_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/KhalidMohomud/ecomApi/internal/domain/entity"
	"github.com/KhalidMohomud/ecomApi/internal/domain/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Reuses newTestRepo's underlying connection setup by opening its
// own — see user_repository_test.go for why config.Load needs
// repoRootEnvFile rather than a bare ".env" here too.
func newTestCategoryRepo(t *testing.T) repository.CategoryRepository {
	t.Helper()
	db := testDB(t)
	return repository.NewCategoryRepository(db)
}

func uniqueSlug(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, uuid.NewString())
}

func TestCategoryRepository_CreateAndGet(t *testing.T) {
	repo := newTestCategoryRepo(t)
	ctx := context.Background()

	category := &entity.Category{Name: "Electronics", Slug: uniqueSlug("electronics")}
	require.NoError(t, repo.Create(ctx, category))
	require.NotEqual(t, uuid.Nil, category.ID)
	t.Cleanup(func() { _ = repo.Delete(ctx, category.ID) })

	byID, err := repo.GetByID(ctx, category.ID)
	require.NoError(t, err)
	require.Equal(t, category.Name, byID.Name)

	bySlug, err := repo.GetBySlug(ctx, category.Slug)
	require.NoError(t, err)
	require.Equal(t, category.ID, bySlug.ID)
}

func TestCategoryRepository_DuplicateSlugReturnsConflict(t *testing.T) {
	repo := newTestCategoryRepo(t)
	ctx := context.Background()
	slug := uniqueSlug("phones")

	first := &entity.Category{Name: "Phones", Slug: slug}
	require.NoError(t, repo.Create(ctx, first))
	t.Cleanup(func() { _ = repo.Delete(ctx, first.ID) })

	second := &entity.Category{Name: "Phones Again", Slug: slug}
	err := repo.Create(ctx, second)

	require.ErrorIs(t, err, entity.ErrConflict)
}

func TestCategoryRepository_ParentChildRelationship(t *testing.T) {
	repo := newTestCategoryRepo(t)
	ctx := context.Background()

	parent := &entity.Category{Name: "Electronics", Slug: uniqueSlug("electronics")}
	require.NoError(t, repo.Create(ctx, parent))
	t.Cleanup(func() { _ = repo.Delete(ctx, parent.ID) })

	child := &entity.Category{Name: "Phones", Slug: uniqueSlug("phones"), ParentID: &parent.ID}
	require.NoError(t, repo.Create(ctx, child))
	t.Cleanup(func() { _ = repo.Delete(ctx, child.ID) })

	fetched, err := repo.GetByID(ctx, child.ID)
	require.NoError(t, err)
	require.NotNil(t, fetched.ParentID)
	require.Equal(t, parent.ID, *fetched.ParentID)
}

func TestCategoryRepository_DeletingParentSetsChildParentIDToNull(t *testing.T) {
	repo := newTestCategoryRepo(t)
	ctx := context.Background()

	parent := &entity.Category{Name: "Electronics", Slug: uniqueSlug("electronics")}
	require.NoError(t, repo.Create(ctx, parent))

	child := &entity.Category{Name: "Phones", Slug: uniqueSlug("phones"), ParentID: &parent.ID}
	require.NoError(t, repo.Create(ctx, child))
	t.Cleanup(func() { _ = repo.Delete(ctx, child.ID) })

	// This is the ON DELETE SET NULL behavior declared in the
	// migration, not application code — deleting the parent row at
	// the database level (bypassing GORM's soft delete, since a hard
	// delete is what actually exercises the FK action) promotes the
	// child to top-level instead of leaving a dangling reference.
	require.NoError(t, repo.Delete(ctx, parent.ID))

	fetched, err := repo.GetByID(ctx, child.ID)
	require.NoError(t, err)
	require.Nil(t, fetched.ParentID, "child's parent_id should be nulled out, not left dangling, when the parent is deleted")
}

func TestCategoryRepository_GetByID_NotFound(t *testing.T) {
	repo := newTestCategoryRepo(t)
	_, err := repo.GetByID(context.Background(), uuid.New())
	require.ErrorIs(t, err, entity.ErrNotFound)
}

func TestCategoryRepository_UpdateAndSoftDelete(t *testing.T) {
	repo := newTestCategoryRepo(t)
	ctx := context.Background()

	category := &entity.Category{Name: "Books", Slug: uniqueSlug("books"), IsActive: true}
	require.NoError(t, repo.Create(ctx, category))

	category.Name = "Books & Media"
	category.IsActive = false
	require.NoError(t, repo.Update(ctx, category))

	fetched, err := repo.GetByID(ctx, category.ID)
	require.NoError(t, err)
	require.Equal(t, "Books & Media", fetched.Name)
	require.False(t, fetched.IsActive)

	require.NoError(t, repo.Delete(ctx, category.ID))
	_, err = repo.GetByID(ctx, category.ID)
	require.ErrorIs(t, err, entity.ErrNotFound)
}
