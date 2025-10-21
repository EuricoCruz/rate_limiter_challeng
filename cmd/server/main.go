package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/EuricoCruz/rate_limiter_challeng/internal/adapter/http/middleware"
	redisAdapter "github.com/EuricoCruz/rate_limiter_challeng/internal/adapter/storage/redis"
	"github.com/EuricoCruz/rate_limiter_challeng/internal/infrastructure/config"
	"github.com/EuricoCruz/rate_limiter_challeng/internal/infrastructure/logger"
	infraRedis "github.com/EuricoCruz/rate_limiter_challeng/internal/infrastructure/redis"
	"github.com/EuricoCruz/rate_limiter_challeng/internal/usecase/check_rate_limit"
	"github.com/go-chi/chi/v5"
)

// configAdapter adapta config.Config para implementar middleware.Config
type configAdapter struct {
	*config.Config
}

func (c *configAdapter) GetTokenConfig(token string) (middleware.TokenConfig, bool) {
	cfg, exists := c.Config.GetTokenConfig(token)
	if !exists {
		return middleware.TokenConfig{}, false
	}
	return middleware.TokenConfig{
		Limit:     cfg.Limit,
		Window:    cfg.Window,
		BlockTime: cfg.BlockTime,
	}, true
}

func main() {
	// 1. Setup logger
	logger := logger.New()
	logger.Info("Starting Rate Limiter")

	// 2. Carrega configuração
	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load config", "error", err)
		os.Exit(1)
	}
	logger.Info("Configuration loaded",
		"port", cfg.ServerPort,
		"redis", fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
		"ip_limit", cfg.IPLimit,
		"tokens_configured", len(cfg.TokenConfigs),
	)

	// 3. Conecta Redis
	redisClient, err := infraRedis.NewClient(cfg)
	if err != nil {
		logger.Error("Failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()
	logger.Info("Connected to Redis")

	// 4. Monta camadas (Dependency Injection)

	// Storage layer
	storage := redisAdapter.NewRedisStorage(redisClient)
	logger.Info("Storage layer initialized")

	// Use case layer
	checkRateLimitUC := check_rate_limit.NewUseCase(storage)
	logger.Info("Use case layer initialized")

	// Middleware layer
	cfgAdapter := &configAdapter{Config: cfg}
	rateLimiterMW := middleware.NewRateLimiterMiddleware(checkRateLimitUC, cfgAdapter)
	logger.Info("Middleware layer initialized")

	// 5. Setup HTTP Router
	r := chi.NewRouter()

	// Aplica rate limiter globalmente
	r.Use(rateLimiterMW.Handle)

	// Rotas
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Rate Limiter is running"))
	})

	// 6. HTTP Server
	srv := &http.Server{
		Addr:         ":" + strconv.Itoa(cfg.ServerPort),
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 7. Start server em goroutine
	go func() {
		logger.Info("Server starting", "port", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server error", "error", err)
			os.Exit(1)
		}
	}()

	// 8. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
	}

	logger.Info("Rate Limiter stopped")
}
