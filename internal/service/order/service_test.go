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
	statuses []int
	orders   []repository.Order
	err      error
}

func (r *fakeRepo) CreateOrder(_ context.Context, params repository.CreateOrderParams) (int64, int64, error) {
	r.createIn = params
	return 5001, 9000, r.err
}

func (r *fakeRepo) ListBuyerOrders(_ context.Context, buyerID int64, statuses []int, limit, offset int) ([]repository.Order, error) {
	r.statuses = statuses
	return r.orders, r.err
}

func (r *fakeRepo) RefundOrder(context.Context, int64, int64, string) (int64, error) {
	return 8800, r.err
}

func (r *fakeRepo) ReceiveOrder(context.Context, int64, int64) error {
	return r.err
}

func (r *fakeRepo) ShipProductOrders(context.Context, int64, int64) (int, int, int, error) {
	return 2, 5, 10, r.err
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

	t.Run("商品不可购买", func(t *testing.T) {
		svc := NewService(&fakeRepo{err: repository.ErrProductNotBuyable})
		ctx := auth.WithCurrentUser(context.Background(), auth.CurrentUser{ID: 2001, Role: auth.RoleBuyer})

		_, err := svc.CreateOrder(ctx, CreateOrderInput{ProductID: 3001, Quantity: 2, IdempotencyKey: "order-1"})

		if err != ErrProductNotBuyable {
			t.Fatalf("error = %v, want ErrProductNotBuyable", err)
		}
	})

	t.Run("金额超过上限", func(t *testing.T) {
		svc := NewService(&fakeRepo{err: repository.ErrOrderAmountTooLarge})
		ctx := auth.WithCurrentUser(context.Background(), auth.CurrentUser{ID: 2001, Role: auth.RoleBuyer})

		_, err := svc.CreateOrder(ctx, CreateOrderInput{ProductID: 3001, Quantity: 2, IdempotencyKey: "order-1"})

		if err != ErrInvalidArgument {
			t.Fatalf("error = %v, want ErrInvalidArgument", err)
		}
	})

	t.Run("幂等冲突", func(t *testing.T) {
		svc := NewService(&fakeRepo{err: repository.ErrIdempotencyConflict})
		ctx := auth.WithCurrentUser(context.Background(), auth.CurrentUser{ID: 2001, Role: auth.RoleBuyer})

		_, err := svc.CreateOrder(ctx, CreateOrderInput{ProductID: 3001, Quantity: 2, IdempotencyKey: "order-1"})

		if err != ErrIdempotencyConflict {
			t.Fatalf("error = %v, want ErrIdempotencyConflict", err)
		}
	})
}

func TestServiceListBuyerOrders_BitsUT(t *testing.T) {
	repo := &fakeRepo{orders: []repository.Order{{ID: 1, ProductID: 3001, ProductNameSnapshot: "phone", Quantity: 2, TotalAmountCent: 2000, Status: orderdomain.StatusShipping, CreatedAt: time.Date(2026, 5, 24, 10, 0, 0, 0, time.UTC)}}}
	svc := NewService(repo)
	ctx := auth.WithCurrentUser(context.Background(), auth.CurrentUser{ID: 2001, Role: auth.RoleBuyer})

	got, err := svc.ListBuyerOrders(ctx, 1, 20, nil)

	if err != nil {
		t.Fatalf("ListBuyerOrders() error = %v", err)
	}
	if len(got) != 1 || got[0].StatusName != "SHIPPING" || got[0].CreatedAt == "" {
		t.Fatalf("ListBuyerOrders() = %+v", got)
	}
	if len(repo.statuses) != 2 || repo.statuses[0] != orderdomain.StatusPlacedUnshipped || repo.statuses[1] != orderdomain.StatusShipping {
		t.Fatalf("statuses = %+v", repo.statuses)
	}
}

func TestServiceListBuyerOrdersWithStatus_BitsUT(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo)
	ctx := auth.WithCurrentUser(context.Background(), auth.CurrentUser{ID: 2001, Role: auth.RoleBuyer})
	status := orderdomain.StatusReceived

	if _, err := svc.ListBuyerOrders(ctx, 1, 20, &status); err != nil {
		t.Fatalf("ListBuyerOrders() error = %v", err)
	}
	if len(repo.statuses) != 1 || repo.statuses[0] != orderdomain.StatusReceived {
		t.Fatalf("statuses = %+v", repo.statuses)
	}
}

func TestServiceRefundOrder_BitsUT(t *testing.T) {
	t.Run("退款成功", func(t *testing.T) {
		svc := NewService(&fakeRepo{})
		ctx := auth.WithCurrentUser(context.Background(), auth.CurrentUser{ID: 2001, Role: auth.RoleBuyer})

		got, err := svc.RefundOrder(ctx, 5001, "refund-1")

		if err != nil {
			t.Fatalf("RefundOrder() error = %v", err)
		}
		if got.Status != orderdomain.StatusRefunded || got.BalanceCent != 8800 {
			t.Fatalf("RefundOrder() = %+v", got)
		}
	})

	t.Run("状态不允许退款", func(t *testing.T) {
		svc := NewService(&fakeRepo{err: repository.ErrOrderStatusInvalid})
		ctx := auth.WithCurrentUser(context.Background(), auth.CurrentUser{ID: 2001, Role: auth.RoleBuyer})

		_, err := svc.RefundOrder(ctx, 5001, "refund-1")

		if err != ErrOrderStatusInvalid {
			t.Fatalf("error = %v, want ErrOrderStatusInvalid", err)
		}
	})
}

func TestServiceReceiveOrder_BitsUT(t *testing.T) {
	svc := NewService(&fakeRepo{})
	ctx := auth.WithCurrentUser(context.Background(), auth.CurrentUser{ID: 2001, Role: auth.RoleBuyer})

	if err := svc.ReceiveOrder(ctx, 5001); err != nil {
		t.Fatalf("ReceiveOrder() error = %v", err)
	}
}

func TestServiceShipProductOrders_BitsUT(t *testing.T) {
	t.Run("一键发货成功", func(t *testing.T) {
		svc := NewService(&fakeRepo{})
		ctx := auth.WithCurrentUser(context.Background(), auth.CurrentUser{ID: 1001, Role: auth.RoleSeller})

		got, err := svc.ShipProductOrders(ctx, 3001)

		if err != nil {
			t.Fatalf("ShipProductOrders() error = %v", err)
		}
		if got.ShippedOrderCount != 2 || got.ShippedQuantity != 5 || got.RemainingInventory != 10 {
			t.Fatalf("ShipProductOrders() = %+v", got)
		}
	})

	t.Run("库存不足", func(t *testing.T) {
		svc := NewService(&fakeRepo{err: repository.ErrInventoryNotEnough})
		ctx := auth.WithCurrentUser(context.Background(), auth.CurrentUser{ID: 1001, Role: auth.RoleSeller})

		_, err := svc.ShipProductOrders(ctx, 3001)

		if err != ErrInventoryNotEnough {
			t.Fatalf("error = %v, want ErrInventoryNotEnough", err)
		}
	})
}
