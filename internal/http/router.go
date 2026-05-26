package http

import (
	"net/http"

	"microservice-demo/internal/http/handler"
	"microservice-demo/internal/http/middleware"
)

func NewRouter(mysql handler.Pinger, redis handler.Pinger) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /health", handler.NewHealthHandler(mysql, redis))
	return middleware.RequestID(mux)
}
