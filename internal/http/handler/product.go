package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"microservice-demo/internal/http/response"
	productsvc "microservice-demo/internal/service/product"
)

type ProductService interface {
	CreateProduct(ctx context.Context, input productsvc.CreateProductInput) (productsvc.CreateProductOutput, error)
	UpdateProduct(ctx context.Context, input productsvc.UpdateProductInput) error
	AddInventory(ctx context.Context, productID int64, quantity int) (productsvc.InventoryOutput, error)
	SearchBuyerProducts(ctx context.Context, namePrefix string, page, pageSize int) ([]productsvc.ProductOutput, error)
	ListSellerProducts(ctx context.Context, input productsvc.SellerProductListInput) ([]productsvc.SellerProductOutput, error)
	ListSellerTrend(ctx context.Context, input productsvc.TrendInput) (productsvc.TrendOutput, error)
	DelistProduct(ctx context.Context, productID int64) (productsvc.DelistProductOutput, error)
}

type ProductHandler struct {
	service ProductService
}

type createProductRequest struct {
	ProductName      string `json:"productName"`
	Description      string `json:"description"`
	PriceCent        int64  `json:"priceCent"`
	Status           int    `json:"status"`
	InitialInventory int    `json:"initialInventory"`
}

type updateProductRequest struct {
	ProductName string `json:"productName"`
	Description string `json:"description"`
	PriceCent   int64  `json:"priceCent"`
	Status      int    `json:"status"`
}

type addInventoryRequest struct {
	Quantity int `json:"quantity"`
}

func NewProductHandler(service ProductService) *ProductHandler {
	return &ProductHandler{service: service}
}

func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req createProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, http.StatusBadRequest, "VALIDATION_FAILED", "请求体格式错误")
		return
	}
	output, err := h.service.CreateProduct(r.Context(), productsvc.CreateProductInput{
		ProductName:      req.ProductName,
		Description:      req.Description,
		PriceCent:        req.PriceCent,
		Status:           req.Status,
		InitialInventory: req.InitialInventory,
	})
	if err != nil {
		writeProductError(w, r, err)
		return
	}
	response.JSON(w, r, http.StatusOK, output)
}

func (h *ProductHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	productID, ok := pathInt64(r, "productId")
	if !ok {
		response.Error(w, r, http.StatusBadRequest, "VALIDATION_FAILED", "商品 ID 不合法")
		return
	}
	var req updateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, http.StatusBadRequest, "VALIDATION_FAILED", "请求体格式错误")
		return
	}
	err := h.service.UpdateProduct(r.Context(), productsvc.UpdateProductInput{
		ProductID:   productID,
		ProductName: req.ProductName,
		Description: req.Description,
		PriceCent:   req.PriceCent,
		Status:      req.Status,
	})
	if err != nil {
		writeProductError(w, r, err)
		return
	}
	response.JSON(w, r, http.StatusOK, map[string]bool{"success": true})
}

func (h *ProductHandler) AddInventory(w http.ResponseWriter, r *http.Request) {
	productID, ok := pathInt64(r, "productId")
	if !ok {
		response.Error(w, r, http.StatusBadRequest, "VALIDATION_FAILED", "商品 ID 不合法")
		return
	}
	var req addInventoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, http.StatusBadRequest, "VALIDATION_FAILED", "请求体格式错误")
		return
	}
	output, err := h.service.AddInventory(r.Context(), productID, req.Quantity)
	if err != nil {
		writeProductError(w, r, err)
		return
	}
	response.JSON(w, r, http.StatusOK, output)
}

func (h *ProductHandler) SearchBuyerProducts(w http.ResponseWriter, r *http.Request) {
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
	items, err := h.service.SearchBuyerProducts(r.Context(), r.URL.Query().Get("namePrefix"), page, pageSize)
	if err != nil {
		writeProductError(w, r, err)
		return
	}
	response.JSON(w, r, http.StatusOK, map[string]any{"items": items})
}

func (h *ProductHandler) ListSellerProducts(w http.ResponseWriter, r *http.Request) {
	status, err := queryOptionalInt(r, "status")
	if err != nil {
		response.Error(w, r, http.StatusBadRequest, "VALIDATION_FAILED", "参数校验失败")
		return
	}
	productID, err := queryOptionalInt64(r, "productId")
	if err != nil {
		response.Error(w, r, http.StatusBadRequest, "VALIDATION_FAILED", "参数校验失败")
		return
	}
	shipped, err := queryOptionalBool(r, "shipped")
	if err != nil {
		response.Error(w, r, http.StatusBadRequest, "VALIDATION_FAILED", "参数校验失败")
		return
	}
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
	items, err := h.service.ListSellerProducts(r.Context(), productsvc.SellerProductListInput{
		StartDate:         r.URL.Query().Get("startDate"),
		EndDate:           r.URL.Query().Get("endDate"),
		Status:            status,
		ProductID:         productID,
		ProductNamePrefix: r.URL.Query().Get("productNamePrefix"),
		Shipped:           shipped,
		Page:              page,
		PageSize:          pageSize,
	})
	if err != nil {
		writeProductError(w, r, err)
		return
	}
	response.JSON(w, r, http.StatusOK, map[string]any{"items": items})
}

func (h *ProductHandler) ListSellerTrend(w http.ResponseWriter, r *http.Request) {
	days, err := queryInt(r, "days", 7)
	if err != nil {
		response.Error(w, r, http.StatusBadRequest, "VALIDATION_FAILED", "参数校验失败")
		return
	}
	output, err := h.service.ListSellerTrend(r.Context(), productsvc.TrendInput{
		StartDate: r.URL.Query().Get("startDate"),
		EndDate:   r.URL.Query().Get("endDate"),
		Days:      days,
	})
	if err != nil {
		writeProductError(w, r, err)
		return
	}
	response.JSON(w, r, http.StatusOK, output)
}

func (h *ProductHandler) DelistProduct(w http.ResponseWriter, r *http.Request) {
	productID, ok := pathInt64(r, "productId")
	if !ok {
		response.Error(w, r, http.StatusBadRequest, "VALIDATION_FAILED", "商品 ID 不合法")
		return
	}
	output, err := h.service.DelistProduct(r.Context(), productID)
	if err != nil {
		writeProductError(w, r, err)
		return
	}
	response.JSON(w, r, http.StatusOK, output)
}

func writeProductError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, productsvc.ErrInvalidArgument):
		response.Error(w, r, http.StatusBadRequest, "VALIDATION_FAILED", "参数校验失败")
	case errors.Is(err, productsvc.ErrUnauthenticated):
		response.Error(w, r, http.StatusUnauthorized, "AUTH_TOKEN_INVALID", "登录凭证无效或已过期")
	case errors.Is(err, productsvc.ErrProductNotFound):
		response.Error(w, r, http.StatusNotFound, "RESOURCE_NOT_FOUND", "商品不存在")
	case errors.Is(err, productsvc.ErrProductStatusInvalid):
		response.Error(w, r, http.StatusConflict, "PRODUCT_STATUS_INVALID", "商品状态不允许操作")
	default:
		response.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "服务内部错误")
	}
}

func pathInt64(r *http.Request, name string) (int64, bool) {
	value, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	return value, err == nil && value > 0
}

func queryInt(r *http.Request, name string, fallback int) (int, error) {
	value := r.URL.Query().Get(name)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}

func queryOptionalInt(r *http.Request, name string) (*int, error) {
	value := r.URL.Query().Get(name)
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func queryOptionalInt64(r *http.Request, name string) (*int64, error) {
	value := r.URL.Query().Get(name)
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		if err != nil {
			return nil, err
		}
		return nil, strconv.ErrSyntax
	}
	return &parsed, nil
}

func queryOptionalBool(r *http.Request, name string) (*bool, error) {
	value := strings.ToLower(r.URL.Query().Get(name))
	if value == "" {
		return nil, nil
	}
	switch value {
	case "true", "1":
		parsed := true
		return &parsed, nil
	case "false", "0":
		parsed := false
		return &parsed, nil
	default:
		return nil, strconv.ErrSyntax
	}
}
