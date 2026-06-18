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
	registerWebRoutes(mux)
	return middleware.RequestID(mux)
}

func NewRouterWithAuth(mysql handler.Pinger, redis handler.Pinger, authHandler *handler.AuthHandler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /health", handler.NewHealthHandler(mysql, redis))
	mux.HandleFunc("POST /api/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)
	registerWebRoutes(mux)
	return middleware.RequestID(mux)
}

func NewRouterWithServices(mysql handler.Pinger, redis handler.Pinger, authHandler *handler.AuthHandler, userHandler *handler.UserHandler, productHandler *handler.ProductHandler, orderHandler *handler.OrderHandler, verifier middleware.TokenVerifier) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /health", handler.NewHealthHandler(mysql, redis))
	mux.HandleFunc("POST /api/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)

	buyerAuth := middleware.Auth(verifier, auth.RoleBuyer)
	mux.Handle("GET /api/buyer/profile", buyerAuth(http.HandlerFunc(userHandler.GetBuyerProfile)))
	mux.Handle("PUT /api/buyer/profile", buyerAuth(http.HandlerFunc(userHandler.UpdateBuyerProfile)))
	mux.Handle("POST /api/buyer/balance/recharge", buyerAuth(http.HandlerFunc(userHandler.RechargeBuyerBalance)))
	mux.Handle("GET /api/buyer/products", buyerAuth(http.HandlerFunc(productHandler.SearchBuyerProducts)))
	mux.Handle("POST /api/buyer/orders", buyerAuth(http.HandlerFunc(orderHandler.CreateOrder)))
	mux.Handle("GET /api/buyer/orders", buyerAuth(http.HandlerFunc(orderHandler.ListBuyerOrders)))
	mux.Handle("POST /api/buyer/orders/{orderId}/refund", buyerAuth(http.HandlerFunc(orderHandler.RefundOrder)))
	mux.Handle("POST /api/buyer/orders/{orderId}/receive", buyerAuth(http.HandlerFunc(orderHandler.ReceiveOrder)))

	anyAuth := middleware.Auth(verifier)
	mux.Handle("POST /api/auth/logout", anyAuth(http.HandlerFunc(authHandler.Logout)))

	sellerAuth := middleware.Auth(verifier, auth.RoleSeller)
	mux.Handle("GET /api/seller/profile", sellerAuth(http.HandlerFunc(userHandler.GetSellerProfile)))
	mux.Handle("PUT /api/seller/profile", sellerAuth(http.HandlerFunc(userHandler.UpdateSellerProfile)))
	mux.Handle("GET /api/seller/products", sellerAuth(http.HandlerFunc(productHandler.ListSellerProducts)))
	mux.Handle("GET /api/seller/orders", sellerAuth(http.HandlerFunc(orderHandler.ListSellerOrders)))
	mux.Handle("POST /api/seller/orders/{orderId}/ship", sellerAuth(http.HandlerFunc(orderHandler.ShipSellerOrder)))
	mux.Handle("POST /api/seller/products", sellerAuth(http.HandlerFunc(productHandler.CreateProduct)))
	mux.Handle("PUT /api/seller/products/{productId}", sellerAuth(http.HandlerFunc(productHandler.UpdateProduct)))
	mux.Handle("POST /api/seller/products/{productId}/inventory/add", sellerAuth(http.HandlerFunc(productHandler.AddInventory)))
	mux.Handle("POST /api/seller/products/{productId}/ship-all", sellerAuth(http.HandlerFunc(orderHandler.ShipProductOrders)))
	mux.Handle("POST /api/seller/products/{productId}/delist", sellerAuth(http.HandlerFunc(productHandler.DelistProduct)))
	mux.Handle("GET /api/seller/trends", sellerAuth(http.HandlerFunc(productHandler.ListSellerTrend)))

	registerWebRoutes(mux)
	return middleware.RequestID(mux)
}

func registerWebRoutes(mux *http.ServeMux) {
	fileServer := http.FileServer(http.Dir("web"))
	mux.Handle("GET /", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		fileServer.ServeHTTP(w, r)
	}))
}
