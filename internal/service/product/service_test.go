package productsvc

import (
	"context"
	"database/sql"
	"testing"

	"microservice-demo/internal/domain/auth"
	productdomain "microservice-demo/internal/domain/product"
	"microservice-demo/internal/repository"
)

type fakeRepo struct {
	createIn       repository.CreateProductParams
	updateIn       repository.UpdateProductParams
	filterIn       repository.SellerProductFilter
	products       []repository.Product
	sellerProducts []repository.SellerProduct
	trends         []repository.TrendPoint
	available      int
	delistStatus   int
}

func (r *fakeRepo) CreateProduct(_ context.Context, params repository.CreateProductParams) (int64, error) {
	r.createIn = params
	return 3001, nil
}

func (r *fakeRepo) UpdateProduct(_ context.Context, params repository.UpdateProductParams) error {
	r.updateIn = params
	return nil
}

func (r *fakeRepo) AddInventory(context.Context, int64, int64, int) (int, error) {
	return r.available, nil
}

func (r *fakeRepo) SearchBuyerProducts(context.Context, string, int, int) ([]repository.Product, error) {
	return r.products, nil
}

func (r *fakeRepo) ListSellerProducts(_ context.Context, filter repository.SellerProductFilter) ([]repository.SellerProduct, error) {
	r.filterIn = filter
	return r.sellerProducts, nil
}

func (r *fakeRepo) ListSellerTrend(context.Context, int64, string, string) ([]repository.TrendPoint, error) {
	return r.trends, nil
}

func (r *fakeRepo) DelistProduct(context.Context, int64, int64) (int, error) {
	return r.delistStatus, nil
}

func TestServiceCreateProduct_BitsUT(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo)
	ctx := auth.WithCurrentUser(context.Background(), auth.CurrentUser{ID: 1001, Role: auth.RoleSeller})

	got, err := svc.CreateProduct(ctx, CreateProductInput{ProductName: "手机壳", PriceCent: 1999, Status: productdomain.StatusOnSale, InitialInventory: 10})

	if err != nil {
		t.Fatalf("CreateProduct() error = %v", err)
	}
	if got.ProductID != 3001 || repo.createIn.SellerID != 1001 || repo.createIn.InitialInventory != 10 {
		t.Fatalf("got = %+v, createIn = %+v", got, repo.createIn)
	}
}

func TestServiceCreateProductRejectUnavailableStatus_BitsUT(t *testing.T) {
	svc := NewService(&fakeRepo{})
	ctx := auth.WithCurrentUser(context.Background(), auth.CurrentUser{ID: 1001, Role: auth.RoleSeller})

	_, err := svc.CreateProduct(ctx, CreateProductInput{ProductName: "手机壳", PriceCent: 1999, Status: productdomain.StatusOffShelf})

	if err != ErrInvalidArgument {
		t.Fatalf("error = %v, want ErrInvalidArgument", err)
	}
}

func TestServiceSearchBuyerProducts_BitsUT(t *testing.T) {
	repo := &fakeRepo{products: []repository.Product{{ID: 1, ProductName: "手机壳", PriceCent: 1999, Status: productdomain.StatusPreSale, AvailableQuantity: 0, ShopName: sql.NullString{String: "店铺", Valid: true}}}}
	svc := NewService(repo)
	ctx := auth.WithCurrentUser(context.Background(), auth.CurrentUser{ID: 2001, Role: auth.RoleBuyer})

	got, err := svc.SearchBuyerProducts(ctx, "手机", 1, 20)

	if err != nil {
		t.Fatalf("SearchBuyerProducts() error = %v", err)
	}
	if len(got) != 1 || got[0].StatusName != "PRE_SALE" || got[0].ShopName != "店铺" {
		t.Fatalf("SearchBuyerProducts() = %+v", got)
	}
}

func TestServiceAddInventory_BitsUT(t *testing.T) {
	t.Run("数量非法", func(t *testing.T) {
		svc := NewService(&fakeRepo{})
		ctx := auth.WithCurrentUser(context.Background(), auth.CurrentUser{ID: 1001, Role: auth.RoleSeller})

		_, err := svc.AddInventory(ctx, 3001, 0)

		if err != ErrInvalidArgument {
			t.Fatalf("error = %v, want ErrInvalidArgument", err)
		}
	})

	t.Run("补库存成功", func(t *testing.T) {
		svc := NewService(&fakeRepo{available: 30})
		ctx := auth.WithCurrentUser(context.Background(), auth.CurrentUser{ID: 1001, Role: auth.RoleSeller})

		got, err := svc.AddInventory(ctx, 3001, 20)

		if err != nil {
			t.Fatalf("AddInventory() error = %v", err)
		}
		if got.AvailableQuantity != 30 {
			t.Fatalf("AvailableQuantity = %d, want 30", got.AvailableQuantity)
		}
	})
}

func TestServiceListSellerProducts_BitsUT(t *testing.T) {
	status := productdomain.StatusOnSale
	repo := &fakeRepo{sellerProducts: []repository.SellerProduct{{Product: repository.Product{ID: 1, ProductName: "手机壳", PriceCent: 1999, Status: status, AvailableQuantity: 10}, DealAmountCent: 10000, RefundAmountCent: 1000, RefundRate: 0.1}}}
	svc := NewService(repo)
	ctx := auth.WithCurrentUser(context.Background(), auth.CurrentUser{ID: 1001, Role: auth.RoleSeller})

	got, err := svc.ListSellerProducts(ctx, SellerProductListInput{StartDate: "2026-05-18", EndDate: "2026-05-24", Status: &status, ProductNamePrefix: "手机", Page: 1, PageSize: 20})

	if err != nil {
		t.Fatalf("ListSellerProducts() error = %v", err)
	}
	if repo.filterIn.SellerID != 1001 || repo.filterIn.ProductNamePrefix != "手机" {
		t.Fatalf("filterIn = %+v", repo.filterIn)
	}
	if len(got) != 1 || got[0].DealAmountCent != 10000 || got[0].RefundRate != 0.1 {
		t.Fatalf("ListSellerProducts() = %+v", got)
	}
}

func TestServiceListSellerTrend_BitsUT(t *testing.T) {
	repo := &fakeRepo{trends: []repository.TrendPoint{{Date: "2026-05-19", DealAmountCent: 10000, RefundAmountCent: 1000, RefundRate: 0.1}}}
	svc := NewService(repo)
	ctx := auth.WithCurrentUser(context.Background(), auth.CurrentUser{ID: 1001, Role: auth.RoleSeller})

	got, err := svc.ListSellerTrend(ctx, TrendInput{StartDate: "2026-05-18", EndDate: "2026-05-20"})

	if err != nil {
		t.Fatalf("ListSellerTrend() error = %v", err)
	}
	if len(got.Points) != 3 || got.Points[0].DealAmountCent != 0 || got.Points[1].RefundRate != 0.1 {
		t.Fatalf("ListSellerTrend() = %+v", got)
	}
}

func TestServiceDelistProduct_BitsUT(t *testing.T) {
	svc := NewService(&fakeRepo{delistStatus: productdomain.StatusOffShelf})
	ctx := auth.WithCurrentUser(context.Background(), auth.CurrentUser{ID: 1001, Role: auth.RoleSeller})

	got, err := svc.DelistProduct(ctx, 3001)
	if err != nil {
		t.Fatalf("DelistProduct() error = %v", err)
	}
	if got.Status != productdomain.StatusOffShelf || got.StatusName != "OFF_SHELF" {
		t.Fatalf("DelistProduct() = %+v", got)
	}
}
