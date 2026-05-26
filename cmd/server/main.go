package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"microservice-demo/internal/config"
	httpserver "microservice-demo/internal/http"
	"microservice-demo/internal/infra/mysql"
	redisinfra "microservice-demo/internal/infra/redis"
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

	server := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      httpserver.NewRouter(db, rdb),
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
