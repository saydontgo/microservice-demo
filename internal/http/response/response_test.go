package response

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestJSON_BitsUT(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	rec.Header().Set(requestIDHeader, "req-1")

	JSON(rec, req, http.StatusOK, map[string]string{"status": "ok"})

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"status":"ok"`) || !strings.Contains(body, `"requestId":"req-1"`) {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestError_BitsUT(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	rec.Header().Set(requestIDHeader, "req-1")

	Error(rec, req, http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE", "dependency unavailable")

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status code = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"code":"DEPENDENCY_UNAVAILABLE"`) || !strings.Contains(body, `"requestId":"req-1"`) {
		t.Fatalf("unexpected body: %s", body)
	}
}
