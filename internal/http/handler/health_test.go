package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakePinger struct {
	err error
}

func (p fakePinger) PingContext(context.Context) error {
	return p.err
}

func TestHealthHandlerServeHTTP_BitsUT(t *testing.T) {
	t.Run("依赖全部正常返回200", func(t *testing.T) {
		h := NewHealthHandler(fakePinger{}, fakePinger{})
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status code = %d, want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		if !strings.Contains(body, `"status":"ok"`) || !strings.Contains(body, `"mysql":"ok"`) || !strings.Contains(body, `"redis":"ok"`) {
			t.Fatalf("unexpected body: %s", body)
		}
	})

	t.Run("MySQL异常返回503", func(t *testing.T) {
		h := NewHealthHandler(fakePinger{err: errors.New("mysql down")}, fakePinger{})
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status code = %d, want %d", rec.Code, http.StatusServiceUnavailable)
		}
		body := rec.Body.String()
		if !strings.Contains(body, `"status":"degraded"`) || !strings.Contains(body, `"mysql":"error"`) {
			t.Fatalf("unexpected body: %s", body)
		}
	})
}
