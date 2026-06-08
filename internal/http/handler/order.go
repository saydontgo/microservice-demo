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
	ListBuyerOrders(ctx context.Context, page, pageSize int, status *int) ([]ordersvc.OrderOutput, error)
	RefundOrder(ctx context.Context, orderID int64, idempotencyKey string) (ordersvc.RefundOutput, error)
	ReceiveOrder(ctx context.Context, orderID int64) error
	ShipProductOrders(ctx context.Context, productID int64) (ordersvc.ShipOutput, error)
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
	page, err := queryInt(r, "page", 1)
	if err != nil {
		response.Error(w, r, http.StatusBadRequest, "VALIDATION_FAILED", "参数校验失败")
		return
	}
	pageSize, err := queryInt(r, "pageSize", 20)
	if err != nil {
		response.Error(w, r, http.StatusBadRequest, "VALIDATION_FAILED", "参数校验失败")
		return
	}
	status, err := queryOptionalInt(r, "status")
	if err != nil {
		response.Error(w, r, http.StatusBadRequest, "VALIDATION_FAILED", "参数校验失败")
		return
	}
	items, err := h.service.ListBuyerOrders(r.Context(), page, pageSize, status)
	if err != nil {
		writeOrderError(w, r, err)
		return
	}
	response.JSON(w, r, http.StatusOK, map[string]any{"items": items})
}

func (h *OrderHandler) RefundOrder(w http.ResponseWriter, r *http.Request) {
	orderID, ok := pathInt64(r, "orderId")
	if !ok {
		response.Error(w, r, http.StatusBadRequest, "VALIDATION_FAILED", "订单 ID 不合法")
		return
	}
	output, err := h.service.RefundOrder(r.Context(), orderID, r.Header.Get("Idempotency-Key"))
	if err != nil {
		writeOrderError(w, r, err)
		return
	}
	response.JSON(w, r, http.StatusOK, output)
}

func (h *OrderHandler) ReceiveOrder(w http.ResponseWriter, r *http.Request) {
	orderID, ok := pathInt64(r, "orderId")
	if !ok {
		response.Error(w, r, http.StatusBadRequest, "VALIDATION_FAILED", "订单 ID 不合法")
		return
	}
	if err := h.service.ReceiveOrder(r.Context(), orderID); err != nil {
		writeOrderError(w, r, err)
		return
	}
	response.JSON(w, r, http.StatusOK, map[string]bool{"success": true})
}

func (h *OrderHandler) ShipProductOrders(w http.ResponseWriter, r *http.Request) {
	productID, ok := pathInt64(r, "productId")
	if !ok {
		response.Error(w, r, http.StatusBadRequest, "VALIDATION_FAILED", "商品 ID 不合法")
		return
	}
	output, err := h.service.ShipProductOrders(r.Context(), productID)
	if err != nil {
		writeOrderError(w, r, err)
		return
	}
	response.JSON(w, r, http.StatusOK, output)
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
	case errors.Is(err, ordersvc.ErrOrderStatusInvalid):
		response.Error(w, r, http.StatusConflict, "ORDER_STATUS_INVALID", "订单状态不允许操作")
	case errors.Is(err, ordersvc.ErrOrderNotFound), errors.Is(err, ordersvc.ErrProductNotFound):
		response.Error(w, r, http.StatusNotFound, "RESOURCE_NOT_FOUND", "资源不存在")
	case errors.Is(err, ordersvc.ErrIdempotencyConflict):
		response.Error(w, r, http.StatusConflict, "IDEMPOTENCY_CONFLICT", "幂等键重复但请求内容不同")
	default:
		response.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "服务内部错误")
	}
}
