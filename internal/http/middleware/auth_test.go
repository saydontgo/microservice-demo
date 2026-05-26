package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"microservice-demo/internal/domain/auth"
)

type fakeVerifier struct {
	user auth.CurrentUser
	err  error
}

func (v fakeVerifier) VerifyToken(context.Context, string) (auth.CurrentUser, error) {
	return v.user, v.err
}

func TestAuth_BitsUT(t *testing.T) {
	t.Run("有效买家token允许访问", func(t *testing.T) {
		nextCalled := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
		})
		h := Auth(fakeVerifier{user: auth.CurrentUser{ID: 1, Role: auth.RoleBuyer}}, auth.RoleBuyer)(next)
		req := httptest.NewRequest(http.MethodGet, "/api/buyer/profile", nil)
		req.Header.Set("Authorization", "Bearer token")
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK || !nextCalled {
			t.Fatalf("status = %d, nextCalled = %v", rec.Code, nextCalled)
		}
	})

	t.Run("角色不匹配返回403", func(t *testing.T) {
		h := Auth(fakeVerifier{user: auth.CurrentUser{ID: 1, Role: auth.RoleSeller}}, auth.RoleBuyer)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		req := httptest.NewRequest(http.MethodGet, "/api/buyer/profile", nil)
		req.Header.Set("Authorization", "Bearer token")
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
		}
	})

	t.Run("token无效返回401", func(t *testing.T) {
		h := Auth(fakeVerifier{err: errors.New("bad token")}, auth.RoleBuyer)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		req := httptest.NewRequest(http.MethodGet, "/api/buyer/profile", nil)
		req.Header.Set("Authorization", "Bearer token")
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})
}
