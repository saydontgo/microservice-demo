package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"microservice-demo/internal/http/response"
	authsvc "microservice-demo/internal/service/auth"
)

type AuthService interface {
	Register(ctx context.Context, input authsvc.RegisterInput) (authsvc.RegisterOutput, error)
	Login(ctx context.Context, input authsvc.LoginInput) (authsvc.LoginOutput, error)
}

type AuthHandler struct {
	service AuthService
}

type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
	Profile  struct {
		Nickname        string `json:"nickname"`
		AvatarURL       string `json:"avatarUrl"`
		Phone           string `json:"phone"`
		ShippingAddress string `json:"shippingAddress"`
		RegistrantName  string `json:"registrantName"`
		ShopName        string `json:"shopName"`
		ShopAvatarURL   string `json:"shopAvatarUrl"`
	} `json:"profile"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func NewAuthHandler(service AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, http.StatusBadRequest, "VALIDATION_FAILED", "请求体格式错误")
		return
	}

	output, err := h.service.Register(r.Context(), authsvc.RegisterInput{
		Username:        req.Username,
		Password:        req.Password,
		Role:            req.Role,
		Nickname:        req.Profile.Nickname,
		AvatarURL:       req.Profile.AvatarURL,
		Phone:           req.Profile.Phone,
		ShippingAddress: req.Profile.ShippingAddress,
		RegistrantName:  req.Profile.RegistrantName,
		ShopName:        req.Profile.ShopName,
		ShopAvatarURL:   req.Profile.ShopAvatarURL,
	})
	if err != nil {
		writeAuthError(w, r, err)
		return
	}
	response.JSON(w, r, http.StatusOK, output)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, http.StatusBadRequest, "VALIDATION_FAILED", "请求体格式错误")
		return
	}

	output, err := h.service.Login(r.Context(), authsvc.LoginInput{Username: req.Username, Password: req.Password, Role: req.Role})
	if err != nil {
		writeAuthError(w, r, err)
		return
	}
	response.JSON(w, r, http.StatusOK, output)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, r, http.StatusOK, map[string]bool{"success": true})
}

func writeAuthError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, authsvc.ErrInvalidArgument):
		response.Error(w, r, http.StatusBadRequest, "VALIDATION_FAILED", "参数校验失败")
	case errors.Is(err, authsvc.ErrUserExists):
		response.Error(w, r, http.StatusConflict, "USER_EXISTS", "用户已存在")
	case errors.Is(err, authsvc.ErrInvalidCredential):
		response.Error(w, r, http.StatusUnauthorized, "AUTH_INVALID_CREDENTIAL", "用户名、密码或身份错误")
	case errors.Is(err, authsvc.ErrRoleMismatch):
		response.Error(w, r, http.StatusForbidden, "AUTH_ROLE_MISMATCH", "登录身份与账号身份不一致")
	case errors.Is(err, authsvc.ErrUserDisabled):
		response.Error(w, r, http.StatusForbidden, "AUTH_FORBIDDEN", "账号已禁用")
	default:
		response.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "服务内部错误")
	}
}
