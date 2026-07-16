package service_test

import (
	"context"
	"testing"

	"github.com/KhalidMohomud/ecomApi/internal/dto"
	"github.com/KhalidMohomud/ecomApi/internal/service"
	"github.com/stretchr/testify/require"
)

func newTestBrandService() (*fakeBrandRepository, service.BrandService) {
	repo := newFakeBrandRepository()
	return repo, service.NewBrandService(repo)
}

func TestBrandService_Create_AutoGeneratesSlugFromName(t *testing.T) {
	_, svc := newTestBrandService()

	resp, err := svc.Create(context.Background(), dto.CreateBrandRequest{Name: "Nike Inc."})

	require.NoError(t, err)
	require.Equal(t, "nike-inc", resp.Slug)
}

func TestBrandService_Create_NormalizesProvidedSlug(t *testing.T) {
	_, svc := newTestBrandService()

	resp, err := svc.Create(context.Background(), dto.CreateBrandRequest{Name: "Nike", Slug: "NIKE"})

	require.NoError(t, err)
	require.Equal(t, "nike", resp.Slug)
}

func TestBrandService_Create_DuplicateSlugReturnsConflict(t *testing.T) {
	_, svc := newTestBrandService()
	ctx := context.Background()

	_, err := svc.Create(ctx, dto.CreateBrandRequest{Name: "Nike"})
	require.NoError(t, err)

	_, err = svc.Create(ctx, dto.CreateBrandRequest{Name: "Nike"})
	require.Error(t, err)
}

func TestBrandService_Update_ChangesFieldsIncludingZeroValue(t *testing.T) {
	_, svc := newTestBrandService()
	ctx := context.Background()

	created, err := svc.Create(ctx, dto.CreateBrandRequest{Name: "Nike"})
	require.NoError(t, err)

	updated, err := svc.Update(ctx, created.ID, dto.UpdateBrandRequest{Name: "Nike Updated", IsActive: false})

	require.NoError(t, err)
	require.Equal(t, "Nike Updated", updated.Name)
	require.False(t, updated.IsActive, "IsActive:false must be persisted, not silently dropped")
}

func TestBrandService_GetBySlug(t *testing.T) {
	_, svc := newTestBrandService()
	ctx := context.Background()
	created, err := svc.Create(ctx, dto.CreateBrandRequest{Name: "Adidas"})
	require.NoError(t, err)

	fetched, err := svc.GetBySlug(ctx, created.Slug)

	require.NoError(t, err)
	require.Equal(t, created.ID, fetched.ID)
}
