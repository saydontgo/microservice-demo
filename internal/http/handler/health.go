package handler

import (
	"context"
	"net/http"
	"time"

	"microservice-demo/internal/http/response"
)

type Pinger interface {
	PingContext(ctx context.Context) error
}

type HealthHandler struct {
	mysql Pinger
	redis Pinger
}

type HealthStatus struct {
	Status string `json:"status"`
	MySQL  string `json:"mysql"`
	Redis  string `json:"redis"`
}

func NewHealthHandler(mysql Pinger, redis Pinger) http.Handler {
	return &HealthHandler{mysql: mysql, redis: redis}
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second)
	defer cancel()

	status := HealthStatus{Status: "ok", MySQL: "ok", Redis: "ok"}
	code := http.StatusOK
	if err := h.mysql.PingContext(ctx); err != nil {
		status.Status = "degraded"
		status.MySQL = "error"
		code = http.StatusServiceUnavailable
	}
	if err := h.redis.PingContext(ctx); err != nil {
		status.Status = "degraded"
		status.Redis = "error"
		code = http.StatusServiceUnavailable
	}

	response.JSON(w, r, code, status)
}
