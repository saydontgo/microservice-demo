package http

import (
	"net/http"

	"microservice-demo/internal/domain/auth"
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

func NewRouterWithServices(mysql handler.Pinger, redis handler.Pinger, authHandler *handler.AuthHandler, userHandler *handler.UserHandler, productHandler *handler.ProductHandler, verifier middleware.TokenVerifier) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /health", handler.NewHealthHandler(mysql, redis))
	mux.HandleFunc("POST /api/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)

	buyerAuth := middleware.Auth(verifier, auth.RoleBuyer)
	mux.Handle("GET /api/buyer/profile", buyerAuth(http.HandlerFunc(userHandler.GetBuyerProfile)))
	mux.Handle("PUT /api/buyer/profile", buyerAuth(http.HandlerFunc(userHandler.UpdateBuyerProfile)))
	mux.Handle("POST /api/buyer/balance/recharge", buyerAuth(http.HandlerFunc(userHandler.RechargeBuyerBalance)))
	mux.Handle("GET /api/buyer/products", buyerAuth(http.HandlerFunc(productHandler.SearchBuyerProducts)))

	sellerAuth := middleware.Auth(verifier, auth.RoleSeller)
	mux.Handle("GET /api/seller/profile", sellerAuth(http.HandlerFunc(userHandler.GetSellerProfile)))
	mux.Handle("PUT /api/seller/profile", sellerAuth(http.HandlerFunc(userHandler.UpdateSellerProfile)))
	mux.Handle("POST /api/seller/products", sellerAuth(http.HandlerFunc(productHandler.CreateProduct)))
	mux.Handle("PUT /api/seller/products/{productId}", sellerAuth(http.HandlerFunc(productHandler.UpdateProduct)))
	mux.Handle("POST /api/seller/products/{productId}/inventory/add", sellerAuth(http.HandlerFunc(productHandler.AddInventory)))

	return middleware.RequestID(mux)
}
