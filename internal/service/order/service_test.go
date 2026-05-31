package ordersvc

import (
	"context"
	"testing"
	"time"

	"microservice-demo/internal/domain/auth"
	orderdomain "microservice-demo/internal/domain/order"
	"microservice-demo/internal/repository"
)

type fakeRepo struct {
	createIn repository.CreateOrderParams
	orders   []repository.Order
	err      error
}

func (r *fakeRepo) CreateOrder(_ context.Context, params repository.CreateOrderParams) (int64, int64, error) {
	r.createIn = params
	return 5001, 9000, r.err
}

func (r *fakeRepo) ListBuyerOrders(_ context.Context, buyerID int64, statuses []int, limit, offset int) ([]repository.Order, error) {
	return r.orders, r.err
}

func TestServiceCreateOrder_BitsUT(t *testing.T) {
	t.Run("下单成功", func(t *testing.T) {
		repo := &fakeRepo{}
		svc := NewService(repo)
		ctx := auth.WithCurrentUser(context.Background(), auth.CurrentUser{ID: 2001, Role: auth.RoleBuyer})

		got, err := svc.CreateOrder(ctx, CreateOrderInput{ProductID: 3001, Quantity: 2, IdempotencyKey: "order-1"})

		if err != nil {
			t.Fatalf("CreateOrder() error = %v", err)
		}
		if got.OrderID != 5001 || got.Status != orderdomain.StatusPlacedUnshipped || repo.createIn.BuyerID != 2001 {
			t.Fatalf("got = %+v, createIn = %+v", got, repo.createIn)
		}
	})

	t.Run("数量超过上限", func(t *testing.T) {
		svc := NewService(&fakeRepo{})
		ctx := auth.WithCurrentUser(context.Background(), auth.CurrentUser{ID: 2001, Role: auth.RoleBuyer})

		_, err := svc.CreateOrder(ctx, CreateOrderInput{ProductID: 3001, Quantity: 101, IdempotencyKey: "order-1"})

		if err != ErrInvalidArgument {
			t.Fatalf("error = %v, want ErrInvalidArgument", err)
		}
	})

	t.Run("余额不足", func(t *testing.T) {
		svc := NewService(&fakeRepo{err: repository.ErrBalanceNotEnough})
		ctx := auth.WithCurrentUser(context.Background(), auth.CurrentUser{ID: 2001, Role: auth.RoleBuyer})

		_, err := svc.CreateOrder(ctx, CreateOrderInput{ProductID: 3001, Quantity: 2, IdempotencyKey: "order-1"})

		if err != ErrBalanceNotEnough {
			t.Fatalf("error = %v, want ErrBalanceNotEnough", err)
		}
	})
}

func TestServiceListBuyerOrders_BitsUT(t *testing.T) {
	repo := &fakeRepo{orders: []repository.Order{{ID: 1, ProductID: 3001, ProductNameSnapshot: "phone", Quantity: 2, TotalAmountCent: 2000, Status: orderdomain.StatusShipping, CreatedAt: time.Date(2026, 5, 24, 10, 0, 0, 0, time.UTC)}}}
	svc := NewService(repo)
	ctx := auth.WithCurrentUser(context.Background(), auth.CurrentUser{ID: 2001, Role: auth.RoleBuyer})

	got, err := svc.ListBuyerOrders(ctx, 1, 20)

	if err != nil {
		t.Fatalf("ListBuyerOrders() error = %v", err)
	}
	if len(got) != 1 || got[0].StatusName != "SHIPPING" || got[0].CreatedAt == "" {
		t.Fatalf("ListBuyerOrders() = %+v", got)
	}
}
