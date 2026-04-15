package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"nyx/internal/agentruntime"
	"nyx/internal/bootstrap"
	"nyx/internal/config"
	"nyx/internal/executor"
	"nyx/internal/functions"
	"nyx/internal/observability"
	"nyx/internal/openai"
	"nyx/internal/orchestrator"
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
		Service: "nyx-orchestrator",
		Format:  cfg.LogFormat,
		Level:   cfg.LogLevel,
	})
	metrics := observability.NewRegistry()
	if err := cfg.Validate("orchestrator"); err != nil {
		return fmt.Errorf("validate orchestrator config: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	bootstrap.ConfigureEmbeddings(cfg)

	repo, err := store.OpenRepository(ctx, cfg.DatabaseURL, store.PoolConfig{
		MaxOpenConns:    cfg.DBMaxOpenConns,
		MaxIdleConns:    cfg.DBMaxIdleConns,
		ConnMaxLifetime: cfg.DBConnMaxLifetime,
		ConnMaxIdleTime: cfg.DBConnMaxIdleTime,
	})
	if err != nil {
		return fmt.Errorf("open repository: %w", err)
	}
	defer func() { _ = repo.Close() }()

	transport, err := queue.OpenTransport(ctx, cfg)
	if err != nil {
		return fmt.Errorf("open transport: %w", err)
	}
	defer func() { _ = transport.Close() }()

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
	runtimeOptions := make([]agentruntime.Option, 0, 2)
	if strings.TrimSpace(cfg.OpenAIModel) != "" {
		runtimeOptions = append(runtimeOptions, agentruntime.WithPromptLibrary(agentruntime.DefaultPromptLibrary(cfg.OpenAIModel)))
	}
	switch strings.ToLower(strings.TrimSpace(cfg.AgentRuntimeMode)) {
	case "openai":
		client := openai.NewClient(openai.ClientConfig{
			APIKey:          cfg.OpenAIAPIKey,
			BaseURL:         cfg.OpenAIBaseURL,
			Model:           cfg.OpenAIModel,
			ReasoningEffort: cfg.OpenAIReasoningEffort,
			MaxOutputTokens: cfg.OpenAIMaxOutputTokens,
		})
		runtimeOptions = append(runtimeOptions, agentruntime.WithPlanner(client), agentruntime.WithActionPolicy(client))
	case "auto":
		if strings.TrimSpace(cfg.OpenAIAPIKey) != "" {
			client := openai.NewClient(openai.ClientConfig{
				APIKey:          cfg.OpenAIAPIKey,
				BaseURL:         cfg.OpenAIBaseURL,
				Model:           cfg.OpenAIModel,
				ReasoningEffort: cfg.OpenAIReasoningEffort,
				MaxOutputTokens: cfg.OpenAIMaxOutputTokens,
			})
			runtimeOptions = append(runtimeOptions, agentruntime.WithPlanner(client), agentruntime.WithActionPolicy(client))
		}
	}
	engine := orchestrator.New(repo, gateway, cfg.PollInterval, cfg.ActionResultTimeout, transport, logger, metrics, cfg.RequireRiskyApproval, runtimeOptions...)
	metrics.SetGauge("nyx_service_info", map[string]string{"service": "nyx-orchestrator", "queue_mode": transport.Mode()}, 1)
	observability.StartServer(ctx, cfg.OrchestratorObserveAddr, logger, observability.Health{
		Service: "nyx-orchestrator",
		Checks: map[string]observability.CheckFunc{
			"repository": repo.Ping,
		},
	}, metrics)

	logger.Info("orchestrator starting", "poll_interval", cfg.PollInterval.String(), "observe_addr", cfg.OrchestratorObserveAddr, "version", version.String(), "agent_runtime_mode", cfg.AgentRuntimeMode, "openai_model", cfg.OpenAIModel)
	if err := engine.RunForever(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("orchestrator exited: %w", err)
	}
	return nil
}
