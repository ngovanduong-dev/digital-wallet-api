package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ngovanduong-dev/digital-wallet-api/internal/config"
	"github.com/ngovanduong-dev/digital-wallet-api/internal/db"
	"github.com/ngovanduong-dev/digital-wallet-api/internal/handler"
	"github.com/ngovanduong-dev/digital-wallet-api/internal/middleware"
	"github.com/ngovanduong-dev/digital-wallet-api/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("create connection pool: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("ping database: %v", err)
	}
	log.Println("connected to database")

	queries := db.New(pool)

	authService := service.NewAuthService(queries, cfg.JWTSecret)
	walletService := service.NewWalletService(queries)
	transferService := service.NewTransferService(queries, pool)

	authHandler := handler.NewAuthHandler(authService)
	walletHandler := handler.NewWalletHandler(walletService)
	transferHandler := handler.NewTransferHandler(transferService)

	r := chi.NewRouter()

	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	r.Post("/auth/register", authHandler.Register)
	r.Post("/auth/login", authHandler.Login)

	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(cfg.JWTSecret))

		r.Get("/wallets/me", walletHandler.GetWallet)
		r.Post("/wallets/deposit", walletHandler.Deposit)
		r.Post("/wallets/withdraw", walletHandler.Withdraw)
		r.Get("/transactions", walletHandler.ListTransactions)
		r.Post("/transfers", transferHandler.Transfer)
	})

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("server starting on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
