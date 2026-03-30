package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"nyx/internal/auth"
	"nyx/internal/config"
	"nyx/internal/executor"
	"nyx/internal/functions"
	"nyx/internal/httpapi"
	"nyx/internal/memvec"
	"nyx/internal/observability"
	"nyx/internal/openai"
	"nyx/internal/queue"
	"nyx/internal/services/browser"
	"nyx/internal/services/memory"
	"nyx/internal/services/search"
	"nyx/internal/store"
	"nyx/internal/version"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()
	logger := observability.NewLogger(observability.LoggerConfig{
		Service: cfg.ServiceName,
		Format:  cfg.LogFormat,
		Level:   cfg.LogLevel,
	})
	metrics := observability.NewRegistry()
	if err := cfg.Validate("api"); err != nil {
		return fmt.Errorf("validate api config: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	configureEmbeddings(cfg)

	repo, err := store.OpenRepository(ctx, cfg.DatabaseURL, store.PoolConfig{
		MaxOpenConns:    cfg.DBMaxOpenConns,
		MaxIdleConns:    cfg.DBMaxIdleConns,
		ConnMaxLifetime: cfg.DBConnMaxLifetime,
		ConnMaxIdleTime: cfg.DBConnMaxIdleTime,
	})
	if err != nil {
		return fmt.Errorf("open repository: %w", err)
	}
	defer repo.Close()

	transport, err := queue.OpenTransport(ctx, cfg)
	if err != nil {
		return fmt.Errorf("open transport: %w", err)
	}
	defer transport.Close()

	memoryService := memory.New(repo)
	browserService := browser.NewServiceWithRuntime(browser.RuntimeConfig{
		Mode:           cfg.BrowserMode,
		Timeout:        cfg.BrowserTimeout,
		ArtifactsRoot:  cfg.BrowserArtifactsRoot,
		Headless:       cfg.BrowserHeadless,
		ExecutablePath: cfg.BrowserExecutablePath,
	})
	searchService := search.NewServiceWithConfig(search.Config{
		Mode:              cfg.SearchMode,
		WebMode:           cfg.SearchWebMode,
		DeepMode:          cfg.SearchDeepMode,
		ExploitMode:       cfg.SearchExploitMode,
		CodeMode:          cfg.SearchCodeMode,
		BaseURL:           cfg.SearchBaseURL,
		Timeout:           cfg.SearchTimeout,
		ResultLimit:       cfg.SearchResultLimit,
		UserAgent:         cfg.SearchUserAgent,
		TavilyAPIKey:      cfg.SearchTavilyAPIKey,
		TavilyBaseURL:     cfg.SearchTavilyBaseURL,
		PerplexityAPIKey:  cfg.SearchPerplexityAPIKey,
		PerplexityBaseURL: cfg.SearchPerplexityBaseURL,
		PerplexityModel:   cfg.SearchPerplexityModel,
		SploitusBaseURL:   cfg.SearchSploitusBaseURL,
	})
	execManager, err := executor.Open(cfg)
	if err != nil {
		return fmt.Errorf("open executor manager: %w", err)
	}
	gateway := functions.NewGateway(browserService, memoryService, execManager, functions.WithSearchService(searchService))
	var options []httpapi.Option
	if strings.TrimSpace(cfg.SupabaseURL) != "" {
		authenticator, err := auth.NewSupabaseAuthenticator(ctx, cfg.SupabaseURL, cfg.SupabaseJWTAudience)
		if err != nil {
			return fmt.Errorf("configure supabase auth: %w", err)
		}
		options = append(options, httpapi.WithAuthenticator(authenticator))
	}
	server := httpapi.NewServer(cfg, repo, gateway, transport, logger, metrics, options...)
	metrics.SetGauge("nyx_service_info", map[string]string{"service": cfg.ServiceName, "queue_mode": transport.Mode()}, 1)
	observability.StartServer(ctx, cfg.APIObserveAddr, logger, observability.Health{
		Service: cfg.ServiceName + "-observability",
		Checks: map[string]observability.CheckFunc{
			"repository": repo.Ping,
		},
	}, metrics)

	httpServer := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      server.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		errCh <- httpServer.ListenAndServe()
	}()

	logger.Info("api starting", "listen_addr", cfg.ListenAddr, "observe_addr", cfg.APIObserveAddr, "version", version.String())
	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("api server exited: %w", err)
		}
		return nil
	case <-ctx.Done():
		logger.Info("api shutdown requested", "signal", ctx.Err())
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("shutdown api server: %w", err)
	}
	if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("api server exited: %w", err)
	}
	return nil
}

func configureEmbeddings(cfg config.Config) {
	switch cfg.MemoryEmbeddingsMode {
	case "openai":
		memvec.Configure(openai.NewEmbeddingProvider(openai.EmbeddingConfig{
			APIKey:     cfg.OpenAIAPIKey,
			BaseURL:    cfg.OpenAIBaseURL,
			Model:      cfg.OpenAIEmbeddingModel,
			Dimensions: cfg.OpenAIEmbeddingDims,
		}))
	case "auto":
		if cfg.OpenAIAPIKey != "" {
			memvec.Configure(openai.NewEmbeddingProvider(openai.EmbeddingConfig{
				APIKey:     cfg.OpenAIAPIKey,
				BaseURL:    cfg.OpenAIBaseURL,
				Model:      cfg.OpenAIEmbeddingModel,
				Dimensions: cfg.OpenAIEmbeddingDims,
			}))
			return
		}
		memvec.Configure(memvec.NewHashProvider(cfg.OpenAIEmbeddingDims))
	default:
		memvec.Configure(memvec.NewHashProvider(cfg.OpenAIEmbeddingDims))
	}
}
