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
	createIn  repository.CreateProductParams
	updateIn  repository.UpdateProductParams
	products  []repository.Product
	available int
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
