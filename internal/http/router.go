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

func NewRouterWithAuth(mysql handler.Pinger, redis handler.Pinger, authHandler *handler.AuthHandler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /health", handler.NewHealthHandler(mysql, redis))
	mux.HandleFunc("POST /api/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)
	return middleware.RequestID(mux)
}
