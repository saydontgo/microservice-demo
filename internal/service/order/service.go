package ordersvc

import (
	"context"
	"errors"
	"strings"

	"microservice-demo/internal/domain/auth"
	orderdomain "microservice-demo/internal/domain/order"
	"microservice-demo/internal/repository"
)

var (
	ErrUnauthenticated    = errors.New("unauthenticated")
	ErrInvalidArgument    = errors.New("invalid argument")
	ErrBalanceNotEnough   = errors.New("balance not enough")
	ErrInventoryNotEnough = errors.New("inventory not enough")
	ErrProductNotBuyable  = errors.New("product not buyable")
)

type Repository interface {
	CreateOrder(ctx context.Context, params repository.CreateOrderParams) (int64, int64, error)
	ListBuyerOrders(ctx context.Context, buyerID int64, statuses []int, limit, offset int) ([]repository.Order, error)
}

type Service struct {
	repo Repository
}

type CreateOrderInput struct {
	ProductID      int64
	Quantity       int
	IdempotencyKey string
}

type CreateOrderOutput struct {
	OrderID         int64 `json:"orderId"`
	Status          int   `json:"status"`
	TotalAmountCent int64 `json:"totalAmountCent,omitempty"`
	BalanceCent     int64 `json:"balanceCent"`
}

type OrderOutput struct {
	OrderID             int64  `json:"orderId"`
	ProductID           int64  `json:"productId"`
	ProductNameSnapshot string `json:"productNameSnapshot"`
	Quantity            int    `json:"quantity"`
	TotalAmountCent     int64  `json:"totalAmountCent"`
	RefundAmountCent    int64  `json:"refundAmountCent"`
	Status              int    `json:"status"`
	StatusName          string `json:"statusName"`
	CreatedAt           string `json:"createdAt"`
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateOrder(ctx context.Context, input CreateOrderInput) (CreateOrderOutput, error) {
	user, err := currentUser(ctx, auth.RoleBuyer)
	if err != nil {
		return CreateOrderOutput{}, err
	}
	if input.ProductID <= 0 || input.Quantity <= 0 || input.Quantity > orderdomain.MaxBuyQuantity || strings.TrimSpace(input.IdempotencyKey) == "" {
		return CreateOrderOutput{}, ErrInvalidArgument
	}
	orderID, balance, err := s.repo.CreateOrder(ctx, repository.CreateOrderParams{
		BuyerID:        user.ID,
		ProductID:      input.ProductID,
		Quantity:       input.Quantity,
		IdempotencyKey: input.IdempotencyKey,
	})
	if errors.Is(err, repository.ErrBalanceNotEnough) {
		return CreateOrderOutput{}, ErrBalanceNotEnough
	}
	if errors.Is(err, repository.ErrInventoryNotEnough) {
		return CreateOrderOutput{}, ErrInventoryNotEnough
	}
	if err != nil {
		return CreateOrderOutput{}, err
	}
	return CreateOrderOutput{OrderID: orderID, Status: orderdomain.StatusPlacedUnshipped, BalanceCent: balance}, nil
}

func (s *Service) ListBuyerOrders(ctx context.Context, page, pageSize int) ([]OrderOutput, error) {
	user, err := currentUser(ctx, auth.RoleBuyer)
	if err != nil {
		return nil, err
	}
	if page <= 0 || pageSize <= 0 || pageSize > 100 {
		return nil, ErrInvalidArgument
	}
	items, err := s.repo.ListBuyerOrders(ctx, user.ID, []int{orderdomain.StatusPlacedUnshipped, orderdomain.StatusShipping}, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, err
	}
	outputs := make([]OrderOutput, 0, len(items))
	for _, item := range items {
		outputs = append(outputs, OrderOutput{
			OrderID:             item.ID,
			ProductID:           item.ProductID,
			ProductNameSnapshot: item.ProductNameSnapshot,
			Quantity:            item.Quantity,
			TotalAmountCent:     item.TotalAmountCent,
			RefundAmountCent:    item.RefundAmountCent,
			Status:              item.Status,
			StatusName:          orderdomain.StatusName(item.Status),
			CreatedAt:           item.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return outputs, nil
}

func currentUser(ctx context.Context, role string) (auth.CurrentUser, error) {
	user, ok := auth.CurrentUserFromContext(ctx)
	if !ok || user.Role != role {
		return auth.CurrentUser{}, ErrUnauthenticated
	}
	return user, nil
}
