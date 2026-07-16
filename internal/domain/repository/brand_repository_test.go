package repository_test

import (
	"context"
	"testing"

	"github.com/KhalidMohomud/ecomApi/internal/domain/entity"
	"github.com/KhalidMohomud/ecomApi/internal/domain/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// BrandRepository is structurally identical to CategoryRepository
// minus the parent/child relationship — same Create/Get/Update/Delete
// pattern already proven in category_repository_test.go and, before
// that, user_repository_test.go. This is a smoke test, not a full
// re-derivation of every edge case: it confirms the wiring (real
// table, real slug uniqueness, real soft delete) works for this
// entity specifically, rather than re-testing behavior the pattern
// already covers.
func TestBrandRepository_CreateGetUpdateDelete(t *testing.T) {
	repo := repository.NewBrandRepository(testDB(t))
	ctx := context.Background()

	brand := &entity.Brand{Name: "Acme", Slug: uniqueSlug("acme")}
	require.NoError(t, repo.Create(ctx, brand))
	require.NotEqual(t, uuid.Nil, brand.ID)

	bySlug, err := repo.GetBySlug(ctx, brand.Slug)
	require.NoError(t, err)
	require.Equal(t, brand.ID, bySlug.ID)

	brand.Name = "Acme Corp"
	brand.IsActive = false
	require.NoError(t, repo.Update(ctx, brand))

	fetched, err := repo.GetByID(ctx, brand.ID)
	require.NoError(t, err)
	require.Equal(t, "Acme Corp", fetched.Name)
	require.False(t, fetched.IsActive)

	require.NoError(t, repo.Delete(ctx, brand.ID))
	_, err = repo.GetByID(ctx, brand.ID)
	require.ErrorIs(t, err, entity.ErrNotFound)
}

func TestBrandRepository_DuplicateSlugReturnsConflict(t *testing.T) {
	repo := repository.NewBrandRepository(testDB(t))
	ctx := context.Background()
	slug := uniqueSlug("nike")

	first := &entity.Brand{Name: "Nike", Slug: slug}
	require.NoError(t, repo.Create(ctx, first))
	t.Cleanup(func() { _ = repo.Delete(ctx, first.ID) })

	second := &entity.Brand{Name: "Nike Imposter", Slug: slug}
	err := repo.Create(ctx, second)
	require.ErrorIs(t, err, entity.ErrConflict)
}
