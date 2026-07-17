package service_test

import (
	"context"
	"testing"

	"github.com/KhalidMohomud/ecomApi/internal/domain/entity"
	"github.com/KhalidMohomud/ecomApi/internal/dto"
	"github.com/KhalidMohomud/ecomApi/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// newTestProductService wires ProductService against three fakes at
// once (product, category, brand) — it needs all three because
// Create/Update validate that CategoryID (and BrandID, if given)
// actually exist before writing anything.
func newTestProductService() (*fakeProductRepository, *fakeCategoryRepository, *fakeBrandRepository, service.ProductService) {
	products := newFakeProductRepository()
	categories := newFakeCategoryRepository()
	brands := newFakeBrandRepository()
	return products, categories, brands, service.NewProductService(products, categories, brands)
}

func seedCategory(t *testing.T, repo *fakeCategoryRepository, name string) uuid.UUID {
	t.Helper()
	c := &entity.Category{Name: name, Slug: name}
	require.NoError(t, repo.Create(context.Background(), c))
	return c.ID
}

func seedBrand(t *testing.T, repo *fakeBrandRepository, name string) uuid.UUID {
	t.Helper()
	b := &entity.Brand{Name: name, Slug: name}
	require.NoError(t, repo.Create(context.Background(), b))
	return b.ID
}

func TestProductService_Create_AutoGeneratesSlugFromName(t *testing.T) {
	_, categories, _, svc := newTestProductService()
	categoryID := seedCategory(t, categories, "shoes")

	resp, err := svc.Create(context.Background(), dto.CreateProductRequest{
		Name:       "Men's Running Shoes",
		SKU:        "SKU-001",
		PriceCents: 1999,
		CategoryID: categoryID,
	})

	require.NoError(t, err)
	require.Equal(t, "mens-running-shoes", resp.Slug)
	require.Equal(t, int64(1999), resp.PriceCents)
	require.InDelta(t, 19.99, resp.Price, 0.001, "Price is PriceCents/100, derived for display")
}

func TestProductService_Create_RejectsNonexistentCategory(t *testing.T) {
	_, _, _, svc := newTestProductService()

	_, err := svc.Create(context.Background(), dto.CreateProductRequest{
		Name:       "Shoe",
		SKU:        "SKU-001",
		PriceCents: 1000,
		CategoryID: uuid.New(),
	})

	require.ErrorIs(t, err, service.ErrInvalidCategory)
}

func TestProductService_Create_RejectsNonexistentBrand(t *testing.T) {
	_, categories, _, svc := newTestProductService()
	categoryID := seedCategory(t, categories, "shoes")
	fakeBrandID := uuid.New()

	_, err := svc.Create(context.Background(), dto.CreateProductRequest{
		Name:       "Shoe",
		SKU:        "SKU-001",
		PriceCents: 1000,
		CategoryID: categoryID,
		BrandID:    &fakeBrandID,
	})

	require.ErrorIs(t, err, service.ErrInvalidBrand)
}

func TestProductService_Create_ValidBrandSucceeds(t *testing.T) {
	_, categories, brands, svc := newTestProductService()
	categoryID := seedCategory(t, categories, "shoes")
	brandID := seedBrand(t, brands, "nike")

	resp, err := svc.Create(context.Background(), dto.CreateProductRequest{
		Name:       "Air Max",
		SKU:        "SKU-001",
		PriceCents: 1000,
		CategoryID: categoryID,
		BrandID:    &brandID,
	})

	require.NoError(t, err)
	require.NotNil(t, resp.BrandID)
	require.Equal(t, brandID, *resp.BrandID)
}

func TestProductService_Create_DuplicateSKUReturnsConflict(t *testing.T) {
	_, categories, _, svc := newTestProductService()
	categoryID := seedCategory(t, categories, "shoes")
	ctx := context.Background()

	req := dto.CreateProductRequest{Name: "Shoe A", SKU: "SKU-DUP", PriceCents: 1000, CategoryID: categoryID}
	_, err := svc.Create(ctx, req)
	require.NoError(t, err)

	req2 := dto.CreateProductRequest{Name: "Shoe B", SKU: "SKU-DUP", PriceCents: 2000, CategoryID: categoryID}
	_, err = svc.Create(ctx, req2)
	require.ErrorIs(t, err, entity.ErrConflict)
}

func TestProductService_Update_ChangesCategoryAndClearsBrand(t *testing.T) {
	_, categories, brands, svc := newTestProductService()
	ctx := context.Background()
	categoryA := seedCategory(t, categories, "shoes")
	categoryB := seedCategory(t, categories, "boots")
	brandID := seedBrand(t, brands, "nike")

	created, err := svc.Create(ctx, dto.CreateProductRequest{
		Name: "Item", SKU: "SKU-001", PriceCents: 1000, CategoryID: categoryA, BrandID: &brandID,
	})
	require.NoError(t, err)

	updated, err := svc.Update(ctx, created.ID, dto.UpdateProductRequest{
		Name: "Item", SKU: "SKU-001", PriceCents: 1500, CategoryID: categoryB, BrandID: nil, IsActive: true,
	})

	require.NoError(t, err)
	require.Equal(t, categoryB, updated.CategoryID)
	require.Nil(t, updated.BrandID, "explicitly passing BrandID: nil in the update must clear it")
	require.Equal(t, int64(1500), updated.PriceCents)
}

func TestProductService_List_ForwardsFilterAndComputesPagination(t *testing.T) {
	_, categories, _, svc := newTestProductService()
	ctx := context.Background()
	categoryID := seedCategory(t, categories, "shoes")

	for i := 0; i < 3; i++ {
		_, err := svc.Create(ctx, dto.CreateProductRequest{
			Name: "Item " + uuid.NewString(), SKU: uuid.NewString(), PriceCents: 1000, CategoryID: categoryID,
		})
		require.NoError(t, err)
	}

	resp, err := svc.List(ctx, dto.ProductListQuery{
		PaginationQuery: dto.PaginationQuery{Page: 1, PageSize: 2},
	})

	require.NoError(t, err)
	require.Len(t, resp.Items, 2)
	require.Equal(t, int64(3), resp.Total)
	require.Equal(t, 2, resp.TotalPages)
}
