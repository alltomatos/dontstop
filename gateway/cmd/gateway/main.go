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
	"github.com/alltomatos/dontstop/gateway/internal/platform/config"
	"github.com/alltomatos/dontstop/gateway/internal/platform/database"
	"github.com/alltomatos/dontstop/gateway/internal/platform/health"
	"github.com/alltomatos/dontstop/gateway/internal/provider"
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
	if cfg.DatabaseURL != "" {
		dbCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		pool, err := database.Open(dbCtx, cfg.DatabaseURL)
		cancel()
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer pool.Close()
		readiness.Register("postgres", pool.Ping)
		deps.Models = provider.NewCatalog(pool)
		deps.Verifier = auth.NewVerifier(auth.NewPostgresRepository(pool))
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
