package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authsvc "microservice-demo/internal/service/auth"
)

type fakeAuthService struct {
	loginOutput authsvc.LoginOutput
	loginErr    error
}

func (s fakeAuthService) Register(context.Context, authsvc.RegisterInput) (authsvc.RegisterOutput, error) {
	return authsvc.RegisterOutput{UserID: 1, Role: "BUYER"}, nil
}

func (s fakeAuthService) Login(context.Context, authsvc.LoginInput) (authsvc.LoginOutput, error) {
	return s.loginOutput, s.loginErr
}

func TestAuthHandlerLogin_BitsUT(t *testing.T) {
	t.Run("登录成功", func(t *testing.T) {
		h := NewAuthHandler(fakeAuthService{loginOutput: authsvc.LoginOutput{Token: "token", ExpiresIn: 7200, Role: "BUYER"}})
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"buyer1","password":"pwd","role":"BUYER"}`))
		rec := httptest.NewRecorder()

		h.Login(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status code = %d, want %d", rec.Code, http.StatusOK)
		}
		if !strings.Contains(rec.Body.String(), `"token":"token"`) {
			t.Fatalf("unexpected body: %s", rec.Body.String())
		}
	})

	t.Run("密码错误", func(t *testing.T) {
		h := NewAuthHandler(fakeAuthService{loginErr: authsvc.ErrInvalidCredential})
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"buyer1","password":"bad","role":"BUYER"}`))
		rec := httptest.NewRecorder()

		h.Login(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status code = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
		if !strings.Contains(rec.Body.String(), "AUTH_INVALID_CREDENTIAL") {
			t.Fatalf("unexpected body: %s", rec.Body.String())
		}
	})
}
