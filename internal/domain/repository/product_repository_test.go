package repository_test

import (
	"context"
	"testing"

	"github.com/KhalidMohomud/ecomApi/internal/domain/entity"
	"github.com/KhalidMohomud/ecomApi/internal/domain/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// testCategoryForProducts creates a throwaway category so product
// tests have a valid (required) category_id to reference, and
// registers its own cleanup.
func testCategoryForProducts(t *testing.T, categoryRepo repository.CategoryRepository) *entity.Category {
	t.Helper()
	category := &entity.Category{Name: "Test Category", Slug: uniqueSlug("test-category")}
	require.NoError(t, categoryRepo.Create(context.Background(), category))
	t.Cleanup(func() { _ = categoryRepo.Delete(context.Background(), category.ID) })
	return category
}

func TestProductRepository_CreateAndGet(t *testing.T) {
	db := testDB(t)
	productRepo := repository.NewProductRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	ctx := context.Background()
	category := testCategoryForProducts(t, categoryRepo)

	product := &entity.Product{
		Name:       "Air Max 90",
		Slug:       uniqueSlug("air-max-90"),
		SKU:        uniqueSlug("sku"),
		PriceCents: 12999,
		CategoryID: category.ID,
	}
	require.NoError(t, productRepo.Create(ctx, product))
	require.NotEqual(t, uuid.Nil, product.ID)
	t.Cleanup(func() { _ = productRepo.Delete(ctx, product.ID) })

	byID, err := productRepo.GetByID(ctx, product.ID)
	require.NoError(t, err)
	require.Equal(t, int64(12999), byID.PriceCents)

	bySlug, err := productRepo.GetBySlug(ctx, product.Slug)
	require.NoError(t, err)
	require.Equal(t, product.ID, bySlug.ID)
}

func TestProductRepository_DuplicateSKUReturnsConflict(t *testing.T) {
	db := testDB(t)
	productRepo := repository.NewProductRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	ctx := context.Background()
	category := testCategoryForProducts(t, categoryRepo)
	sku := uniqueSlug("sku")

	first := &entity.Product{Name: "A", Slug: uniqueSlug("a"), SKU: sku, PriceCents: 100, CategoryID: category.ID}
	require.NoError(t, productRepo.Create(ctx, first))
	t.Cleanup(func() { _ = productRepo.Delete(ctx, first.ID) })

	second := &entity.Product{Name: "B", Slug: uniqueSlug("b"), SKU: sku, PriceCents: 200, CategoryID: category.ID}
	err := productRepo.Create(ctx, second)
	require.ErrorIs(t, err, entity.ErrConflict)
}

func TestProductRepository_ExistsByCategoryID(t *testing.T) {
	db := testDB(t)
	productRepo := repository.NewProductRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	ctx := context.Background()
	category := testCategoryForProducts(t, categoryRepo)

	empty, err := productRepo.ExistsByCategoryID(ctx, category.ID)
	require.NoError(t, err)
	require.False(t, empty, "a fresh category should have no products")

	product := &entity.Product{Name: "A", Slug: uniqueSlug("a"), SKU: uniqueSlug("sku"), PriceCents: 100, CategoryID: category.ID}
	require.NoError(t, productRepo.Create(ctx, product))
	t.Cleanup(func() { _ = productRepo.Delete(ctx, product.ID) })

	nonEmpty, err := productRepo.ExistsByCategoryID(ctx, category.ID)
	require.NoError(t, err)
	require.True(t, nonEmpty)
}

func TestProductRepository_List_FiltersByCategoryPriceAndSearch(t *testing.T) {
	db := testDB(t)
	productRepo := repository.NewProductRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	ctx := context.Background()
	categoryA := testCategoryForProducts(t, categoryRepo)
	categoryB := testCategoryForProducts(t, categoryRepo)

	cheap := &entity.Product{Name: "Cheap Shoe", Slug: uniqueSlug("cheap"), SKU: uniqueSlug("sku"), PriceCents: 1000, CategoryID: categoryA.ID}
	expensive := &entity.Product{Name: "Expensive Shoe", Slug: uniqueSlug("expensive"), SKU: uniqueSlug("sku"), PriceCents: 9000, CategoryID: categoryA.ID}
	otherCategory := &entity.Product{Name: "Other Category Item", Slug: uniqueSlug("other"), SKU: uniqueSlug("sku"), PriceCents: 5000, CategoryID: categoryB.ID}
	for _, p := range []*entity.Product{cheap, expensive, otherCategory} {
		require.NoError(t, productRepo.Create(ctx, p))
		// Go 1.22+ scopes the loop variable `p` per-iteration, so a
		// plain closure here already captures the right product on
		// each pass — no need for the IIFE-to-capture-by-value trick
		// older Go code needed to avoid every closure sharing one
		// final loop variable value.
		t.Cleanup(func() { _ = productRepo.Delete(ctx, p.ID) })
	}

	t.Run("filters by category", func(t *testing.T) {
		items, total, err := productRepo.List(ctx, repository.ProductFilter{CategoryID: &categoryA.ID}, 0, 10)
		require.NoError(t, err)
		require.Equal(t, int64(2), total)
		require.Len(t, items, 2)
	})

	t.Run("filters by price range", func(t *testing.T) {
		min := int64(5000)
		items, total, err := productRepo.List(ctx, repository.ProductFilter{CategoryID: &categoryA.ID, MinPriceCents: &min}, 0, 10)
		require.NoError(t, err)
		require.Equal(t, int64(1), total)
		require.Equal(t, "Expensive Shoe", items[0].Name)
	})

	t.Run("searches name case-insensitively", func(t *testing.T) {
		items, total, err := productRepo.List(ctx, repository.ProductFilter{Search: "cheap"}, 0, 10)
		require.NoError(t, err)
		require.Equal(t, int64(1), total)
		require.Equal(t, "Cheap Shoe", items[0].Name)
	})

	t.Run("sorts by price ascending", func(t *testing.T) {
		items, _, err := productRepo.List(ctx, repository.ProductFilter{CategoryID: &categoryA.ID, Sort: repository.ProductSortPriceAsc}, 0, 10)
		require.NoError(t, err)
		require.Len(t, items, 2)
		require.Equal(t, "Cheap Shoe", items[0].Name)
		require.Equal(t, "Expensive Shoe", items[1].Name)
	})

	t.Run("sorts by price descending", func(t *testing.T) {
		items, _, err := productRepo.List(ctx, repository.ProductFilter{CategoryID: &categoryA.ID, Sort: repository.ProductSortPriceDesc}, 0, 10)
		require.NoError(t, err)
		require.Len(t, items, 2)
		require.Equal(t, "Expensive Shoe", items[0].Name)
		require.Equal(t, "Cheap Shoe", items[1].Name)
	})
}

func TestProductRepository_GetByID_NotFound(t *testing.T) {
	productRepo := repository.NewProductRepository(testDB(t))
	_, err := productRepo.GetByID(context.Background(), uuid.New())
	require.ErrorIs(t, err, entity.ErrNotFound)
}

func TestProductRepository_UpdateAndSoftDelete(t *testing.T) {
	db := testDB(t)
	productRepo := repository.NewProductRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	ctx := context.Background()
	category := testCategoryForProducts(t, categoryRepo)

	product := &entity.Product{Name: "Original", Slug: uniqueSlug("original"), SKU: uniqueSlug("sku"), PriceCents: 1000, CategoryID: category.ID, IsActive: true}
	require.NoError(t, productRepo.Create(ctx, product))

	product.Name = "Updated"
	product.IsActive = false
	require.NoError(t, productRepo.Update(ctx, product))

	fetched, err := productRepo.GetByID(ctx, product.ID)
	require.NoError(t, err)
	require.Equal(t, "Updated", fetched.Name)
	require.False(t, fetched.IsActive)

	require.NoError(t, productRepo.Delete(ctx, product.ID))
	_, err = productRepo.GetByID(ctx, product.ID)
	require.ErrorIs(t, err, entity.ErrNotFound)
}
