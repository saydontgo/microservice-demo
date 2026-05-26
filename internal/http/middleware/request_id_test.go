package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestID_BitsUT(t *testing.T) {
	t.Run("复用请求头中的request id", func(t *testing.T) {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.Header.Set(RequestIDHeader, "req-1")
		rec := httptest.NewRecorder()

		RequestID(next).ServeHTTP(rec, req)

		if rec.Header().Get(RequestIDHeader) != "req-1" {
			t.Fatalf("request id = %q, want req-1", rec.Header().Get(RequestIDHeader))
		}
	})

	t.Run("没有request id时自动生成", func(t *testing.T) {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()

		RequestID(next).ServeHTTP(rec, req)

		if rec.Header().Get(RequestIDHeader) == "" {
			t.Fatal("request id should not be empty")
		}
	})
}
