package ordersvc

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"microservice-demo/internal/domain/auth"
	orderdomain "microservice-demo/internal/domain/order"
	"microservice-demo/internal/repository"
)

var (
	ErrUnauthenticated     = errors.New("unauthenticated")
	ErrInvalidArgument     = errors.New("invalid argument")
	ErrBalanceNotEnough    = errors.New("balance not enough")
	ErrInventoryNotEnough  = errors.New("inventory not enough")
	ErrProductNotBuyable   = errors.New("product not buyable")
	ErrOrderStatusInvalid  = errors.New("order status invalid")
	ErrOrderNotFound       = errors.New("order not found")
	ErrProductNotFound     = errors.New("product not found")
	ErrIdempotencyConflict = errors.New("idempotency conflict")
)

type Repository interface {
	CreateOrder(ctx context.Context, params repository.CreateOrderParams) (int64, int64, error)
	ListBuyerOrders(ctx context.Context, buyerID int64, statuses []int, limit, offset int) ([]repository.Order, error)
	ListSellerOrders(ctx context.Context, filter repository.SellerOrderFilter) ([]repository.SellerOrder, error)
	RefundOrder(ctx context.Context, buyerID, orderID int64, idempotencyKey string) (int64, error)
	ReceiveOrder(ctx context.Context, buyerID, orderID int64) error
	ShipSellerOrder(ctx context.Context, sellerID, orderID int64) (int64, int, int, error)
	ShipProductOrders(ctx context.Context, sellerID, productID int64) (int, int, int, error)
}

type Service struct {
	repo Repository
}

type CreateOrderInput struct {
	ProductID      int64
	Quantity       int
	IdempotencyKey string
}

type SellerOrderListInput struct {
	ProductID         *int64
	ProductNamePrefix string
	Page              int
	PageSize          int
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

type SellerOrderOutput struct {
	OrderID             int64  `json:"orderId"`
	BuyerID             int64  `json:"buyerId"`
	ProductID           int64  `json:"productId"`
	ProductNameSnapshot string `json:"productNameSnapshot"`
	Quantity            int    `json:"quantity"`
	TotalAmountCent     int64  `json:"totalAmountCent"`
	RefundAmountCent    int64  `json:"refundAmountCent"`
	Status              int    `json:"status"`
	StatusName          string `json:"statusName"`
	CreatedAt           string `json:"createdAt"`
	ShippedAt           string `json:"shippedAt,omitempty"`
	ReceivedAt          string `json:"receivedAt,omitempty"`
	RefundedAt          string `json:"refundedAt,omitempty"`
}

type RefundOutput struct {
	OrderID     int64 `json:"orderId"`
	Status      int   `json:"status"`
	BalanceCent int64 `json:"balanceCent"`
}

type ShipOutput struct {
	ProductID          int64 `json:"productId"`
	ShippedOrderCount  int   `json:"shippedOrderCount"`
	ShippedQuantity    int   `json:"shippedQuantity"`
	RemainingInventory int   `json:"remainingInventory"`
}

type ShipOrderOutput struct {
	OrderID            int64 `json:"orderId"`
	ProductID          int64 `json:"productId"`
	Status             int   `json:"status"`
	ShippedQuantity    int   `json:"shippedQuantity"`
	RemainingInventory int   `json:"remainingInventory"`
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
	if errors.Is(err, repository.ErrProductNotBuyable) {
		return CreateOrderOutput{}, ErrProductNotBuyable
	}
	if errors.Is(err, repository.ErrOrderAmountTooLarge) {
		return CreateOrderOutput{}, ErrInvalidArgument
	}
	if errors.Is(err, repository.ErrIdempotencyConflict) {
		return CreateOrderOutput{}, ErrIdempotencyConflict
	}
	if err != nil {
		return CreateOrderOutput{}, err
	}
	return CreateOrderOutput{OrderID: orderID, Status: orderdomain.StatusPlacedUnshipped, BalanceCent: balance}, nil
}

func (s *Service) ListBuyerOrders(ctx context.Context, page, pageSize int, status *int) ([]OrderOutput, error) {
	user, err := currentUser(ctx, auth.RoleBuyer)
	if err != nil {
		return nil, err
	}
	if page <= 0 || pageSize <= 0 || pageSize > 100 {
		return nil, ErrInvalidArgument
	}
	statuses := []int{orderdomain.StatusPlacedUnshipped, orderdomain.StatusShipping}
	if status != nil {
		if !validOrderStatus(*status) {
			return nil, ErrInvalidArgument
		}
		statuses = []int{*status}
	}
	items, err := s.repo.ListBuyerOrders(ctx, user.ID, statuses, pageSize, (page-1)*pageSize)
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

func (s *Service) ListSellerOrders(ctx context.Context, input SellerOrderListInput) ([]SellerOrderOutput, error) {
	user, err := currentUser(ctx, auth.RoleSeller)
	if err != nil {
		return nil, err
	}
	productNamePrefix := strings.TrimSpace(input.ProductNamePrefix)
	if input.Page <= 0 || input.PageSize <= 0 || input.PageSize > 100 {
		return nil, ErrInvalidArgument
	}
	if input.ProductID != nil && *input.ProductID <= 0 {
		return nil, ErrInvalidArgument
	}
	if input.ProductID == nil && productNamePrefix == "" {
		return nil, ErrInvalidArgument
	}
	items, err := s.repo.ListSellerOrders(ctx, repository.SellerOrderFilter{
		SellerID:          user.ID,
		ProductID:         input.ProductID,
		ProductNamePrefix: productNamePrefix,
		Limit:             input.PageSize,
		Offset:            (input.Page - 1) * input.PageSize,
	})
	if err != nil {
		return nil, err
	}
	outputs := make([]SellerOrderOutput, 0, len(items))
	for _, item := range items {
		outputs = append(outputs, SellerOrderOutput{
			OrderID:             item.ID,
			BuyerID:             item.BuyerID,
			ProductID:           item.ProductID,
			ProductNameSnapshot: item.ProductNameSnapshot,
			Quantity:            item.Quantity,
			TotalAmountCent:     item.TotalAmountCent,
			RefundAmountCent:    item.RefundAmountCent,
			Status:              item.Status,
			StatusName:          orderdomain.StatusName(item.Status),
			CreatedAt:           item.CreatedAt.Format("2006-01-02 15:04:05"),
			ShippedAt:           formatNullTime(item.ShippedAt),
			ReceivedAt:          formatNullTime(item.ReceivedAt),
			RefundedAt:          formatNullTime(item.RefundedAt),
		})
	}
	return outputs, nil
}

func (s *Service) RefundOrder(ctx context.Context, orderID int64, idempotencyKey string) (RefundOutput, error) {
	user, err := currentUser(ctx, auth.RoleBuyer)
	if err != nil {
		return RefundOutput{}, err
	}
	if orderID <= 0 || strings.TrimSpace(idempotencyKey) == "" {
		return RefundOutput{}, ErrInvalidArgument
	}
	balance, err := s.repo.RefundOrder(ctx, user.ID, orderID, idempotencyKey)
	if errors.Is(err, repository.ErrOrderStatusInvalid) {
		return RefundOutput{}, ErrOrderStatusInvalid
	}
	if errors.Is(err, repository.ErrIdempotencyConflict) {
		return RefundOutput{}, ErrIdempotencyConflict
	}
	if errors.Is(err, sql.ErrNoRows) {
		return RefundOutput{}, ErrOrderNotFound
	}
	if err != nil {
		return RefundOutput{}, err
	}
	return RefundOutput{OrderID: orderID, Status: orderdomain.StatusRefunded, BalanceCent: balance}, nil
}

func (s *Service) ReceiveOrder(ctx context.Context, orderID int64) error {
	user, err := currentUser(ctx, auth.RoleBuyer)
	if err != nil {
		return err
	}
	if orderID <= 0 {
		return ErrInvalidArgument
	}
	err = s.repo.ReceiveOrder(ctx, user.ID, orderID)
	if errors.Is(err, repository.ErrOrderStatusInvalid) {
		return ErrOrderStatusInvalid
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrOrderNotFound
	}
	return err
}

func (s *Service) ShipSellerOrder(ctx context.Context, orderID int64) (ShipOrderOutput, error) {
	user, err := currentUser(ctx, auth.RoleSeller)
	if err != nil {
		return ShipOrderOutput{}, err
	}
	if orderID <= 0 {
		return ShipOrderOutput{}, ErrInvalidArgument
	}
	productID, quantity, remaining, err := s.repo.ShipSellerOrder(ctx, user.ID, orderID)
	if errors.Is(err, repository.ErrInventoryNotEnough) {
		return ShipOrderOutput{}, ErrInventoryNotEnough
	}
	if errors.Is(err, repository.ErrOrderStatusInvalid) {
		return ShipOrderOutput{}, ErrOrderStatusInvalid
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ShipOrderOutput{}, ErrOrderNotFound
	}
	if err != nil {
		return ShipOrderOutput{}, err
	}
	return ShipOrderOutput{
		OrderID:            orderID,
		ProductID:          productID,
		Status:             orderdomain.StatusShipping,
		ShippedQuantity:    quantity,
		RemainingInventory: remaining,
	}, nil
}

func (s *Service) ShipProductOrders(ctx context.Context, productID int64) (ShipOutput, error) {
	user, err := currentUser(ctx, auth.RoleSeller)
	if err != nil {
		return ShipOutput{}, err
	}
	if productID <= 0 {
		return ShipOutput{}, ErrInvalidArgument
	}
	orderCount, quantity, remaining, err := s.repo.ShipProductOrders(ctx, user.ID, productID)
	if errors.Is(err, repository.ErrInventoryNotEnough) {
		return ShipOutput{}, ErrInventoryNotEnough
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ShipOutput{}, ErrProductNotFound
	}
	if err != nil {
		return ShipOutput{}, err
	}
	return ShipOutput{ProductID: productID, ShippedOrderCount: orderCount, ShippedQuantity: quantity, RemainingInventory: remaining}, nil
}

func currentUser(ctx context.Context, role string) (auth.CurrentUser, error) {
	user, ok := auth.CurrentUserFromContext(ctx)
	if !ok || user.Role != role {
		return auth.CurrentUser{}, ErrUnauthenticated
	}
	return user, nil
}

func validOrderStatus(status int) bool {
	return status == orderdomain.StatusPlacedUnshipped ||
		status == orderdomain.StatusShipping ||
		status == orderdomain.StatusReceived ||
		status == orderdomain.StatusRefunded
}

func formatNullTime(value sql.NullTime) string {
	if !value.Valid {
		return ""
	}
	return value.Time.Format("2006-01-02 15:04:05")
}
