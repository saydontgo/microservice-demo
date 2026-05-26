package response

import (
	"encoding/json"
	"net/http"
)

const requestIDHeader = "X-Request-Id"

type Body struct {
	Data      any    `json:"data,omitempty"`
	Code      string `json:"code,omitempty"`
	Message   string `json:"message,omitempty"`
	RequestID string `json:"requestId,omitempty"`
}

func JSON(w http.ResponseWriter, r *http.Request, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(Body{
		Data:      data,
		RequestID: requestID(w, r),
	})
}

func Error(w http.ResponseWriter, r *http.Request, statusCode int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(Body{
		Code:      code,
		Message:   message,
		RequestID: requestID(w, r),
	})
}

func requestID(w http.ResponseWriter, r *http.Request) string {
	requestID := w.Header().Get(requestIDHeader)
	if requestID != "" {
		return requestID
	}
	return r.Header.Get(requestIDHeader)
}
