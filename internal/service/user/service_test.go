package usersvc

import (
	"context"
	"database/sql"
	"testing"

	"microservice-demo/internal/domain/auth"
	"microservice-demo/internal/repository"
)

type fakeRepo struct {
	buyer      repository.BuyerProfile
	seller     repository.SellerProfile
	buyerIn    repository.UpdateBuyerProfileParams
	sellerIn   repository.UpdateSellerProfileParams
	rechargeID int64
}

func (r *fakeRepo) GetBuyerProfile(context.Context, int64) (repository.BuyerProfile, error) {
	return r.buyer, nil
}

func (r *fakeRepo) UpdateBuyerProfile(_ context.Context, params repository.UpdateBuyerProfileParams) error {
	r.buyerIn = params
	return nil
}

func (r *fakeRepo) RechargeBuyerBalance(context.Context, int64, int64, string) (int64, error) {
	return r.rechargeID, nil
}

func (r *fakeRepo) GetSellerProfile(context.Context, int64) (repository.SellerProfile, error) {
	return r.seller, nil
}

func (r *fakeRepo) UpdateSellerProfile(_ context.Context, params repository.UpdateSellerProfileParams) error {
	r.sellerIn = params
	return nil
}

func TestServiceGetBuyerProfile_BitsUT(t *testing.T) {
	repo := &fakeRepo{buyer: repository.BuyerProfile{UserID: 10, Nickname: "buyer", Phone: sql.NullString{String: "138", Valid: true}, BalanceCent: 100}}
	svc := NewService(repo)
	ctx := auth.WithCurrentUser(context.Background(), auth.CurrentUser{ID: 10, Role: auth.RoleBuyer})

	got, err := svc.GetBuyerProfile(ctx)

	if err != nil {
		t.Fatalf("GetBuyerProfile() error = %v", err)
	}
	if got.UserID != 10 || got.Phone != "138" || got.BalanceCent != 100 {
		t.Fatalf("GetBuyerProfile() = %+v", got)
	}
}

func TestServiceUpdateSellerProfile_BitsUT(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo)
	ctx := auth.WithCurrentUser(context.Background(), auth.CurrentUser{ID: 20, Role: auth.RoleSeller})

	err := svc.UpdateSellerProfile(ctx, UpdateSellerProfileInput{RegistrantName: "张三", ShopName: "张三小店", Theme: "DARK"})

	if err != nil {
		t.Fatalf("UpdateSellerProfile() error = %v", err)
	}
	if repo.sellerIn.UserID != 20 || repo.sellerIn.Theme != "DARK" {
		t.Fatalf("sellerIn = %+v", repo.sellerIn)
	}
}

func TestServiceRechargeBuyerBalance_BitsUT(t *testing.T) {
	t.Run("充值参数合法", func(t *testing.T) {
		repo := &fakeRepo{rechargeID: 500}
		svc := NewService(repo)
		ctx := auth.WithCurrentUser(context.Background(), auth.CurrentUser{ID: 10, Role: auth.RoleBuyer})

		got, err := svc.RechargeBuyerBalance(ctx, 100, "idem-1")

		if err != nil {
			t.Fatalf("RechargeBuyerBalance() error = %v", err)
		}
		if got.BalanceCent != 500 {
			t.Fatalf("BalanceCent = %d, want 500", got.BalanceCent)
		}
	})

	t.Run("缺少幂等键", func(t *testing.T) {
		svc := NewService(&fakeRepo{})
		ctx := auth.WithCurrentUser(context.Background(), auth.CurrentUser{ID: 10, Role: auth.RoleBuyer})

		_, err := svc.RechargeBuyerBalance(ctx, 100, "")

		if err != ErrInvalidArgument {
			t.Fatalf("error = %v, want ErrInvalidArgument", err)
		}
	})
}
