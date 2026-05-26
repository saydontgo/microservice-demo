package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"microservice-demo/internal/http/response"
	usersvc "microservice-demo/internal/service/user"
)

type UserService interface {
	GetBuyerProfile(ctx context.Context) (usersvc.BuyerProfileOutput, error)
	UpdateBuyerProfile(ctx context.Context, input usersvc.UpdateBuyerProfileInput) error
	RechargeBuyerBalance(ctx context.Context, amountCent int64, idempotencyKey string) (usersvc.RechargeOutput, error)
	GetSellerProfile(ctx context.Context) (usersvc.SellerProfileOutput, error)
	UpdateSellerProfile(ctx context.Context, input usersvc.UpdateSellerProfileInput) error
}

type UserHandler struct {
	service UserService
}

type updateBuyerProfileRequest struct {
	Nickname        string `json:"nickname"`
	AvatarURL       string `json:"avatarUrl"`
	Phone           string `json:"phone"`
	ShippingAddress string `json:"shippingAddress"`
}

type rechargeRequest struct {
	AmountCent int64 `json:"amountCent"`
}

type updateSellerProfileRequest struct {
	RegistrantName string `json:"registrantName"`
	ShopName       string `json:"shopName"`
	ShopAvatarURL  string `json:"shopAvatarUrl"`
	Theme          string `json:"theme"`
}

func NewUserHandler(service UserService) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) GetBuyerProfile(w http.ResponseWriter, r *http.Request) {
	profile, err := h.service.GetBuyerProfile(r.Context())
	if err != nil {
		writeUserError(w, r, err)
		return
	}
	response.JSON(w, r, http.StatusOK, profile)
}

func (h *UserHandler) UpdateBuyerProfile(w http.ResponseWriter, r *http.Request) {
	var req updateBuyerProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, http.StatusBadRequest, "VALIDATION_FAILED", "请求体格式错误")
		return
	}
	err := h.service.UpdateBuyerProfile(r.Context(), usersvc.UpdateBuyerProfileInput{
		Nickname:        req.Nickname,
		AvatarURL:       req.AvatarURL,
		Phone:           req.Phone,
		ShippingAddress: req.ShippingAddress,
	})
	if err != nil {
		writeUserError(w, r, err)
		return
	}
	response.JSON(w, r, http.StatusOK, map[string]bool{"success": true})
}

func (h *UserHandler) RechargeBuyerBalance(w http.ResponseWriter, r *http.Request) {
	var req rechargeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, http.StatusBadRequest, "VALIDATION_FAILED", "请求体格式错误")
		return
	}
	output, err := h.service.RechargeBuyerBalance(r.Context(), req.AmountCent, r.Header.Get("Idempotency-Key"))
	if err != nil {
		writeUserError(w, r, err)
		return
	}
	response.JSON(w, r, http.StatusOK, output)
}

func (h *UserHandler) GetSellerProfile(w http.ResponseWriter, r *http.Request) {
	profile, err := h.service.GetSellerProfile(r.Context())
	if err != nil {
		writeUserError(w, r, err)
		return
	}
	response.JSON(w, r, http.StatusOK, profile)
}

func (h *UserHandler) UpdateSellerProfile(w http.ResponseWriter, r *http.Request) {
	var req updateSellerProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, http.StatusBadRequest, "VALIDATION_FAILED", "请求体格式错误")
		return
	}
	err := h.service.UpdateSellerProfile(r.Context(), usersvc.UpdateSellerProfileInput{
		RegistrantName: req.RegistrantName,
		ShopName:       req.ShopName,
		ShopAvatarURL:  req.ShopAvatarURL,
		Theme:          req.Theme,
	})
	if err != nil {
		writeUserError(w, r, err)
		return
	}
	response.JSON(w, r, http.StatusOK, map[string]bool{"success": true})
}

func writeUserError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, usersvc.ErrInvalidArgument):
		response.Error(w, r, http.StatusBadRequest, "VALIDATION_FAILED", "参数校验失败")
	case errors.Is(err, usersvc.ErrUnauthenticated):
		response.Error(w, r, http.StatusUnauthorized, "AUTH_TOKEN_INVALID", "登录凭证无效或已过期")
	case errors.Is(err, usersvc.ErrProfileNotFound):
		response.Error(w, r, http.StatusNotFound, "RESOURCE_NOT_FOUND", "用户资料不存在")
	default:
		response.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "服务内部错误")
	}
}
