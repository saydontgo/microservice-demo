package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"microservice-demo/internal/http/response"
	ordersvc "microservice-demo/internal/service/order"
)

type OrderService interface {
	CreateOrder(ctx context.Context, input ordersvc.CreateOrderInput) (ordersvc.CreateOrderOutput, error)
	ListBuyerOrders(ctx context.Context, page, pageSize int) ([]ordersvc.OrderOutput, error)
}

type OrderHandler struct {
	service OrderService
}

type createOrderRequest struct {
	ProductID int64 `json:"productId"`
	Quantity  int   `json:"quantity"`
}

func NewOrderHandler(service OrderService) *OrderHandler {
	return &OrderHandler{service: service}
}

func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req createOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, http.StatusBadRequest, "VALIDATION_FAILED", "请求体格式错误")
		return
	}
	output, err := h.service.CreateOrder(r.Context(), ordersvc.CreateOrderInput{ProductID: req.ProductID, Quantity: req.Quantity, IdempotencyKey: r.Header.Get("Idempotency-Key")})
	if err != nil {
		writeOrderError(w, r, err)
		return
	}
	response.JSON(w, r, http.StatusOK, output)
}

func (h *OrderHandler) ListBuyerOrders(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListBuyerOrders(r.Context(), queryInt(r, "page", 1), queryInt(r, "pageSize", 20))
	if err != nil {
		writeOrderError(w, r, err)
		return
	}
	response.JSON(w, r, http.StatusOK, map[string]any{"items": items})
}

func writeOrderError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ordersvc.ErrInvalidArgument):
		response.Error(w, r, http.StatusBadRequest, "VALIDATION_FAILED", "参数校验失败")
	case errors.Is(err, ordersvc.ErrUnauthenticated):
		response.Error(w, r, http.StatusUnauthorized, "AUTH_TOKEN_INVALID", "登录凭证无效或已过期")
	case errors.Is(err, ordersvc.ErrBalanceNotEnough):
		response.Error(w, r, http.StatusConflict, "BALANCE_NOT_ENOUGH", "余额不足")
	case errors.Is(err, ordersvc.ErrInventoryNotEnough):
		response.Error(w, r, http.StatusConflict, "INVENTORY_NOT_ENOUGH", "库存不足")
	case errors.Is(err, ordersvc.ErrProductNotBuyable):
		response.Error(w, r, http.StatusConflict, "PRODUCT_NOT_BUYABLE", "商品不可购买")
	default:
		response.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "服务内部错误")
	}
}
