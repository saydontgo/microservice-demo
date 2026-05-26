package middleware

import (
	"net/http"
	"strconv"
	"time"
)

const RequestIDHeader = "X-Request-Id"

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get(RequestIDHeader)
		if requestID == "" {
			requestID = strconv.FormatInt(time.Now().UnixNano(), 10)
		}
		w.Header().Set(RequestIDHeader, requestID)
		next.ServeHTTP(w, r)
	})
}
