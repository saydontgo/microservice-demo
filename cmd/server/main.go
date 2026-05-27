package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"microservice-demo/internal/config"
	httpserver "microservice-demo/internal/http"
	"microservice-demo/internal/http/handler"
	"microservice-demo/internal/infra/mysql"
	redisinfra "microservice-demo/internal/infra/redis"
	"microservice-demo/internal/repository"
	authsvc "microservice-demo/internal/service/auth"
	productsvc "microservice-demo/internal/service/product"
	usersvc "microservice-demo/internal/service/user"
)

func main() {
	cfg := config.Load()

	db, err := mysql.Open(cfg.MySQLDSN())
	if err != nil {
		log.Fatalf("open mysql: %v", err)
	}
	defer db.Close()

	rdb := redisinfra.NewClient(cfg.RedisOptions())
	defer rdb.Close()

	authRepo := repository.NewAuthRepository(db)
	authService := authsvc.NewService(authRepo, cfg.PasswordSalt, cfg.TokenSecret, cfg.TokenTTL)
	authHandler := handler.NewAuthHandler(authService)
	userRepo := repository.NewUserRepository(db)
	userService := usersvc.NewService(userRepo)
	userHandler := handler.NewUserHandler(userService)
	productRepo := repository.NewProductRepository(db)
	productService := productsvc.NewService(productRepo)
	productHandler := handler.NewProductHandler(productService)

	server := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      httpserver.NewRouterWithServices(db, rdb, authHandler, userHandler, productHandler, authService),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Printf("mysql ping failed on startup: %v", err)
	}
	if err := rdb.PingContext(ctx); err != nil {
		log.Printf("redis ping failed on startup: %v", err)
	}

	log.Printf("server listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen: %v", err)
	}
}
