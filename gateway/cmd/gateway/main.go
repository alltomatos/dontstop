package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/api"
	"github.com/alltomatos/dontstop/gateway/internal/auth"
	"github.com/alltomatos/dontstop/gateway/internal/chat"
	"github.com/alltomatos/dontstop/gateway/internal/platform/config"
	"github.com/alltomatos/dontstop/gateway/internal/platform/database"
	"github.com/alltomatos/dontstop/gateway/internal/platform/health"
	redisclient "github.com/alltomatos/dontstop/gateway/internal/platform/redis"
	"github.com/alltomatos/dontstop/gateway/internal/platform/secret"
	"github.com/alltomatos/dontstop/gateway/internal/provider"
	"github.com/alltomatos/dontstop/gateway/internal/routing"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	readinessTimeout = 2 * time.Second
	shutdownTimeout  = 10 * time.Second
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	if err := run(logger); err != nil {
		logger.Error("gateway stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	readiness := health.NewRegistry(readinessTimeout)
	deps := api.Dependencies{Readiness: readiness}
	var pool *pgxpool.Pool
	if cfg.DatabaseURL != "" {
		dbCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		var err error
		pool, err = database.Open(dbCtx, cfg.DatabaseURL)
		cancel()
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer pool.Close()
		readiness.Register("postgres", pool.Ping)
		deps.Models = provider.NewCatalog(pool)
		deps.Verifier = auth.NewVerifier(auth.NewPostgresRepository(pool))
	}
	var redisConn *redis.Client
	if cfg.RedisURL != "" {
		redisCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		var err error
		redisConn, err = redisclient.Open(redisCtx, cfg.RedisURL)
		cancel()
		if err != nil {
			return fmt.Errorf("open Redis: %w", err)
		}
		defer func() { _ = redisConn.Close() }()
		readiness.Register("redis", func(ctx context.Context) error {
			return redisConn.Ping(ctx).Err()
		})
	}
	if pool != nil && redisConn != nil && cfg.MasterKey != "" {
		box, err := secret.NewBoxFromBase64(cfg.MasterKey)
		if err != nil {
			return fmt.Errorf("load master key: %w", err)
		}
		router := routing.NewRouter(routing.NewPostgresRepository(pool), routing.NewRedisBreaker(redisConn, routing.BreakerConfig{}))
		store := chat.NewRedisIdempotencyStore(redisConn, 24*time.Hour)
		deps.Chat = chat.NewService(router, box, provider.NewAdapter, store)
	} else if cfg.DatabaseURL != "" {
		logger.Warn("chat completions disabled", "reason", "DATABASE_URL, REDIS_URL and MASTER_KEY are required")
	}

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           api.NewRouter(deps),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("gateway listening", "port", cfg.Port)
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		logger.Info("shutdown requested")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}
	if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
