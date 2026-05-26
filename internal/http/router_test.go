package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"microservice-demo/internal/http/middleware"
)

type fakePinger struct{}

func (fakePinger) PingContext(context.Context) error {
	return nil
}

func TestNewRouter_BitsUT(t *testing.T) {
	router := NewRouter(fakePinger{}, fakePinger{})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set(middleware.RequestIDHeader, "req-router")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Header().Get(middleware.RequestIDHeader) != "req-router" {
		t.Fatalf("request id = %q, want req-router", rec.Header().Get(middleware.RequestIDHeader))
	}
}
