package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"nyx/internal/config"
	"nyx/internal/executor"
	"nyx/internal/functions"
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
		Service: "nyx-executor",
		Format:  cfg.LogFormat,
		Level:   cfg.LogLevel,
	})
	metrics := observability.NewRegistry()
	if err := cfg.Validate("executor"); err != nil {
		return fmt.Errorf("validate executor config: %w", err)
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

	if transport.Mode() != "jetstream" {
		return errors.New("NATS_URL is required for executor transport")
	}

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
	flowController := executor.NewFlowController(cfg.FlowMaxConcurrentActions, cfg.FlowMinActionInterval)
	gateway := functions.NewGateway(browserService, memoryService, execManager, functions.WithSearchService(searchService))
	metrics.SetGauge("nyx_service_info", map[string]string{"service": "nyx-executor", "queue_mode": transport.Mode()}, 1)
	observability.StartServer(ctx, cfg.ExecutorObserveAddr, logger, observability.Health{
		Service: "nyx-executor",
		Checks: map[string]observability.CheckFunc{
			"repository": repo.Ping,
		},
	}, metrics)

	logger.Info("executor starting", "observe_addr", cfg.ExecutorObserveAddr, "version", version.String())
	err = transport.ConsumeActionRequests(ctx, func(execCtx context.Context, req queue.ActionRequestMessage) (queue.ActionResultMessage, error) {
		metrics.IncCounter("nyx_executor_action_requests_total", map[string]string{"function": req.FunctionName}, 1)
		release, err := flowController.Acquire(execCtx, req.FlowID)
		if err != nil {
			return queue.ActionResultMessage{}, err
		}
		defer release()
		result := gateway.Call(execCtx, req.FlowID, req.ActionID, req.FunctionName, req.Input)
		msg := queue.ActionResultMessage{
			FlowID:       req.FlowID,
			ActionID:     req.ActionID,
			FunctionName: req.FunctionName,
			Profile:      result.Profile,
			Runtime:      result.Runtime,
			Output:       result.Output,
		}
		if result.Err != nil {
			metrics.IncCounter("nyx_executor_action_failures_total", map[string]string{"function": req.FunctionName}, 1)
			msg.Error = result.Err.Error()
		}
		logger.Info("executor action processed", "flow_id", req.FlowID, "action_id", req.ActionID, "function", req.FunctionName, "error", msg.Error)
		return msg, nil
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("executor exited: %w", err)
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
