package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
	"github.com/alltomatos/dontstop/gateway/internal/api"
	"github.com/alltomatos/dontstop/gateway/internal/auth"
	"github.com/alltomatos/dontstop/gateway/internal/cache"
	"github.com/alltomatos/dontstop/gateway/internal/chat"
	mcpruntime "github.com/alltomatos/dontstop/gateway/internal/mcp"
	"github.com/alltomatos/dontstop/gateway/internal/platform/config"
	"github.com/alltomatos/dontstop/gateway/internal/platform/database"
	"github.com/alltomatos/dontstop/gateway/internal/platform/health"
	redisclient "github.com/alltomatos/dontstop/gateway/internal/platform/redis"
	"github.com/alltomatos/dontstop/gateway/internal/platform/secret"
	"github.com/alltomatos/dontstop/gateway/internal/platform/telemetry"
	"github.com/alltomatos/dontstop/gateway/internal/provider"
	proxycoord "github.com/alltomatos/dontstop/gateway/internal/proxy"
	"github.com/alltomatos/dontstop/gateway/internal/routing"
	"github.com/alltomatos/dontstop/gateway/internal/usage"
	"github.com/alltomatos/dontstop/gateway/internal/webshare"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const (
	readinessTimeout = 2 * time.Second
	shutdownTimeout  = 10 * time.Second
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		if err := healthcheck(); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}
	stdio := len(os.Args) > 1 && os.Args[1] == "mcp"
	var output io.Writer = os.Stdout
	if stdio {
		output = os.Stderr
	}
	logger := slog.New(slog.NewJSONHandler(output, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
	if stdio {
		if err := run(logger, true); err != nil {
			logger.Error("gateway mcp stopped", "error", err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	if err := run(logger, false); err != nil {
		logger.Error("gateway stopped", "error", err)
		os.Exit(1)
	}
}

func healthcheck() error {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Get("http://127.0.0.1:" + port + "/readyz")
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("readiness returned %d", response.StatusCode)
	}
	return nil
}

func run(logger *slog.Logger, stdio bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	telemetryProvider, err := telemetry.Setup(context.Background(), "dontstop-gateway", "0.1.0")
	if err != nil {
		return fmt.Errorf("setup telemetry: %w", err)
	}
	defer func() { _ = telemetryProvider.Shutdown(context.Background()) }()

	readiness := health.NewRegistry(readinessTimeout)
	deps := api.Dependencies{Readiness: readiness, MaxChatBody: cfg.MaxChatBodyBytes}
	var pool *pgxpool.Pool
	var providerService *admin.ProviderService
	var webshareService *admin.WebshareService
	var comboService *admin.ComboService
	var usageRecorder *usage.Recorder
	var outbox *usage.OutboxPublisher
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
		deps.AdminAuth = admin.NewAuthService(pool)
		usageRecorder = usage.NewRecorder(pool, provider.NewCatalog(pool))
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
		deps.Redis = redisConn
		readiness.Register("redis", func(ctx context.Context) error {
			return redisConn.Ping(ctx).Err()
		})
	}
	if pool != nil && redisConn != nil && cfg.MasterKey != "" {
		box, err := secret.NewBoxFromBase64(cfg.MasterKey)
		if err != nil {
			return fmt.Errorf("load master key: %w", err)
		}
		providerService = admin.NewProviderService(pool, box)
		providerService.SetAgentRouterRelayURL(cfg.AgentRouterRelayURL)
		deps.Admin = providerService
		deps.AdminProxy = admin.NewProxyService(pool, box)
		webshareService = admin.NewWebshareService(pool, box, webshare.NewClient(nil), redisConn)
		deps.AdminWebshare = webshareService
		deps.AdminCatalog = providerService
		deps.AdminKeys = admin.NewAPIKeyService(pool)
		usageService := admin.NewUsageService(pool)
		deps.AdminUsage = usageService
		mcpAdmin := admin.NewMCPService(pool)
		deps.AdminMCP = mcpAdmin
		deps.AdminCache = admin.NewCacheService(pool)
		deps.AdminDLP = admin.NewDLPService(pool)
		comboService = admin.NewComboService(pool)
		deps.AdminCombos = comboService
		deps.AdminComboAI = admin.NewComboAIService(pool, providerService, comboService)
		deps.AdminEndpoints = admin.NewEndpointService()
		router := routing.NewRouter(routing.NewPostgresRepository(pool), routing.NewRedisBreaker(redisConn, routing.BreakerConfig{}))
		store := chat.NewRedisIdempotencyStore(redisConn, 24*time.Hour)
		coordinator := proxycoord.NewCoordinator(proxycoord.NewPostgresRepository(pool), proxycoord.NewRedisState(redisConn), box)
		coordinator.SetPoolRefresher(webshareService)
		chatService := chat.NewService(router, box, provider.NewAdapter, store, cache.NewRedisExactCache(redisConn), "v1", usageRecorder)
		chatService.SetProxyResolver(coordinator)
		chatService.SetComboResolver(routing.NewComboRepository(pool))
		deps.Chat = chatService
		deps.MCP = mcpruntime.NewService(mcpruntime.Dependencies{
			Store:  mcpAdmin,
			Chat:   chatService,
			Models: deps.Models,
			Usage:  usageService,
			Combos: comboService,
			Logger: logger,
		})
	} else if cfg.DatabaseURL != "" {
		logger.Warn("chat completions disabled", "reason", "DATABASE_URL, REDIS_URL and MASTER_KEY are required")
	}
	if pool != nil && cfg.NATSURL != "" {
		publisher, err := usage.NewNATSPublisher(cfg.NATSURL)
		if err != nil {
			return fmt.Errorf("open NATS publisher: %w", err)
		}
		defer publisher.Close()
		outbox = usage.NewOutboxPublisher(pool, publisher)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if stdio {
		return runMCP(ctx, deps, logger)
	}

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           otelhttp.NewHandler(api.NewRouter(deps), "gateway"),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	if webshareService != nil {
		go webshareService.Run(ctx, cfg.WebshareSyncInterval)
	}

	if providerService != nil {
		go providerService.RunModelTestWorker(ctx, 2*time.Second)
	}

	if outbox != nil {
		go outbox.Run(ctx, time.Second, func(err error) {
			logger.Warn("usage outbox publish failed", "error", err)
		})
	}

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

func runMCP(ctx context.Context, deps api.Dependencies, logger *slog.Logger) error {
	if deps.MCP == nil || deps.Verifier == nil {
		return errors.New("MCP stdio requires DATABASE_URL, REDIS_URL and MASTER_KEY")
	}
	key := strings.TrimSpace(os.Getenv("DONTSTOP_MCP_API_KEY"))
	if key == "" {
		return errors.New("DONTSTOP_MCP_API_KEY is required")
	}
	principal, err := deps.Verifier.Authenticate(ctx, "Bearer "+key)
	if err != nil {
		return fmt.Errorf("authenticate DONTSTOP_MCP_API_KEY: %w", err)
	}
	logger.Info("mcp stdio listening", "tenant_id", principal.TenantID)
	return deps.MCP.RunStdio(ctx, principal)
}
