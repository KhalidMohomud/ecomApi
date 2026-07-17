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

func newTestCategoryService() (*fakeCategoryRepository, service.CategoryService) {
	repo := newFakeCategoryRepository()
	return repo, service.NewCategoryService(repo, newFakeProductRepository())
}

func TestCategoryService_Create_AutoGeneratesSlugFromName(t *testing.T) {
	_, svc := newTestCategoryService()

	resp, err := svc.Create(context.Background(), dto.CreateCategoryRequest{Name: "Men's Running Shoes!"})

	require.NoError(t, err)
	require.Equal(t, "mens-running-shoes", resp.Slug)
}

func TestCategoryService_Create_NormalizesProvidedSlug(t *testing.T) {
	_, svc := newTestCategoryService()

	resp, err := svc.Create(context.Background(), dto.CreateCategoryRequest{Name: "Shoes", Slug: "  Custom SLUG  "})

	require.NoError(t, err)
	require.Equal(t, "custom-slug", resp.Slug)
}

func TestCategoryService_Create_DuplicateSlugReturnsConflict(t *testing.T) {
	_, svc := newTestCategoryService()
	ctx := context.Background()

	_, err := svc.Create(ctx, dto.CreateCategoryRequest{Name: "Shoes"})
	require.NoError(t, err)

	_, err = svc.Create(ctx, dto.CreateCategoryRequest{Name: "Shoes"})
	require.Error(t, err, "same name normalizes to the same slug, so this must conflict")
}

func TestCategoryService_Create_NonexistentParentIsRejected(t *testing.T) {
	_, svc := newTestCategoryService()
	fakeID := uuid.New()

	_, err := svc.Create(context.Background(), dto.CreateCategoryRequest{Name: "Phones", ParentID: &fakeID})

	require.ErrorIs(t, err, service.ErrInvalidParentCategory)
}

func TestCategoryService_Update_CannotBeOwnParent(t *testing.T) {
	_, svc := newTestCategoryService()
	ctx := context.Background()

	created, err := svc.Create(ctx, dto.CreateCategoryRequest{Name: "Electronics"})
	require.NoError(t, err)

	_, err = svc.Update(ctx, created.ID, dto.UpdateCategoryRequest{Name: "Electronics", ParentID: &created.ID})

	require.ErrorIs(t, err, service.ErrInvalidParentCategory)
}

func TestCategoryService_Update_ValidParentSucceeds(t *testing.T) {
	_, svc := newTestCategoryService()
	ctx := context.Background()

	parent, err := svc.Create(ctx, dto.CreateCategoryRequest{Name: "Electronics"})
	require.NoError(t, err)
	child, err := svc.Create(ctx, dto.CreateCategoryRequest{Name: "Phones"})
	require.NoError(t, err)

	updated, err := svc.Update(ctx, child.ID, dto.UpdateCategoryRequest{Name: "Phones", ParentID: &parent.ID, IsActive: true})

	require.NoError(t, err)
	require.NotNil(t, updated.ParentID)
	require.Equal(t, parent.ID, *updated.ParentID)
}

func TestCategoryService_List_Pagination(t *testing.T) {
	_, svc := newTestCategoryService()
	ctx := context.Background()
	for _, name := range []string{"A", "B", "C"} {
		_, err := svc.Create(ctx, dto.CreateCategoryRequest{Name: name})
		require.NoError(t, err)
	}

	resp, err := svc.List(ctx, 1, 2)

	require.NoError(t, err)
	require.Len(t, resp.Items, 2)
	require.Equal(t, int64(3), resp.Total)
	require.Equal(t, 2, resp.TotalPages)
}

func TestCategoryService_Delete_BlockedWhenCategoryHasProducts(t *testing.T) {
	categoryRepo := newFakeCategoryRepository()
	productRepo := newFakeProductRepository()
	svc := service.NewCategoryService(categoryRepo, productRepo)
	ctx := context.Background()

	category, err := svc.Create(ctx, dto.CreateCategoryRequest{Name: "Electronics"})
	require.NoError(t, err)

	product := &entity.Product{Name: "Phone", Slug: "phone", SKU: "sku-1", PriceCents: 1000, CategoryID: category.ID}
	require.NoError(t, productRepo.Create(ctx, product))

	err = svc.Delete(ctx, category.ID)

	require.ErrorIs(t, err, service.ErrCategoryHasProducts)
}

func TestCategoryService_Delete_SucceedsWhenCategoryHasNoProducts(t *testing.T) {
	categoryRepo := newFakeCategoryRepository()
	productRepo := newFakeProductRepository()
	svc := service.NewCategoryService(categoryRepo, productRepo)
	ctx := context.Background()

	category, err := svc.Create(ctx, dto.CreateCategoryRequest{Name: "Electronics"})
	require.NoError(t, err)

	require.NoError(t, svc.Delete(ctx, category.ID))
}
