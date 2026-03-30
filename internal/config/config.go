package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ListenAddr               string
	ServiceName              string
	DatabaseURL              string
	AgentRuntimeMode         string
	PollInterval             time.Duration
	NATSURL                  string
	FlowStream               string
	FlowSubject              string
	FlowConsumer             string
	ActionStream             string
	ActionSubject            string
	ActionConsumer           string
	ActionResultStream       string
	ActionResultSubject      string
	EventStream              string
	EventSubject             string
	DLQStream                string
	DLQSubject               string
	QueueMaxDeliver          int
	ActionResultTimeout      time.Duration
	ExecutorMode             string
	ExecutorImage            string
	ExecutorPentestImage     string
	ExecutorWorkspaceRoot    string
	ExecutorNetworkMode      string
	ExecutorNetworkName      string
	ExecutorEnableNetRaw     bool
	RequireRiskyApproval     bool
	FlowMaxConcurrentActions int
	FlowMinActionInterval    time.Duration
	BrowserMode              string
	BrowserTimeout           time.Duration
	BrowserArtifactsRoot     string
	BrowserHeadless          bool
	BrowserExecutablePath    string
	SearchMode               string
	SearchWebMode            string
	SearchDeepMode           string
	SearchExploitMode        string
	SearchCodeMode           string
	SearchBaseURL            string
	SearchTimeout            time.Duration
	SearchResultLimit        int
	SearchUserAgent          string
	SearchTavilyAPIKey       string
	SearchTavilyBaseURL      string
	SearchPerplexityAPIKey   string
	SearchPerplexityBaseURL  string
	SearchPerplexityModel    string
	SearchSploitusBaseURL    string
	APIKey                   string
	SupabaseURL              string
	SupabaseJWTAudience      string
	DefaultTenant            string
	RequireFlowApproval      bool
	LogFormat                string
	LogLevel                 string
	APIObserveAddr           string
	OrchestratorObserveAddr  string
	ExecutorObserveAddr      string
	HTTPRateLimitRequests    int
	HTTPRateLimitWindow      time.Duration
	OpenAIAPIKey             string
	OpenAIBaseURL            string
	OpenAIModel              string
	OpenAIReasoningEffort    string
	OpenAIMaxOutputTokens    int
	MemoryEmbeddingsMode     string
	OpenAIEmbeddingModel     string
	OpenAIEmbeddingDims      int
	CORSAllowedOrigins       string
	DBMaxOpenConns           int
	DBMaxIdleConns           int
	DBConnMaxLifetime        time.Duration
	DBConnMaxIdleTime        time.Duration
}

func Load() Config {
	return Config{
		ListenAddr:               envOr("NYX_LISTEN_ADDR", ":8080"),
		ServiceName:              envOr("NYX_SERVICE_NAME", "nyx-api"),
		DatabaseURL:              envOr("DATABASE_URL", ""),
		AgentRuntimeMode:         envOr("NYX_AGENT_RUNTIME_MODE", "auto"),
		PollInterval:             durationOr("NYX_POLL_INTERVAL", 1500*time.Millisecond),
		NATSURL:                  envOr("NATS_URL", ""),
		FlowStream:               envOr("NYX_FLOW_STREAM", "NYX_FLOW_RUNS"),
		FlowSubject:              envOr("NYX_FLOW_SUBJECT", "nyx.flows.run"),
		FlowConsumer:             envOr("NYX_FLOW_CONSUMER", "nyx-orchestrator"),
		ActionStream:             envOr("NYX_ACTION_STREAM", "NYX_ACTION_REQUESTS"),
		ActionSubject:            envOr("NYX_ACTION_SUBJECT", "nyx.actions.execute"),
		ActionConsumer:           envOr("NYX_ACTION_CONSUMER", "nyx-executor"),
		ActionResultStream:       envOr("NYX_ACTION_RESULT_STREAM", "NYX_ACTION_RESULTS"),
		ActionResultSubject:      envOr("NYX_ACTION_RESULT_SUBJECT", "nyx.actions.result"),
		EventStream:              envOr("NYX_EVENT_STREAM", "NYX_EVENTS"),
		EventSubject:             envOr("NYX_EVENT_SUBJECT", "nyx.events.flow"),
		DLQStream:                envOr("NYX_DLQ_STREAM", "NYX_DLQ"),
		DLQSubject:               envOr("NYX_DLQ_SUBJECT", "nyx.dlq"),
		QueueMaxDeliver:          intOr("NYX_QUEUE_MAX_DELIVER", 3),
		ActionResultTimeout:      durationOr("NYX_ACTION_RESULT_TIMEOUT", 30*time.Second),
		ExecutorMode:             envOr("NYX_EXECUTOR_MODE", "local"),
		ExecutorImage:            envOr("NYX_EXECUTOR_IMAGE", "alpine:3.20"),
		ExecutorPentestImage:     envOr("NYX_EXECUTOR_IMAGE_FOR_PENTEST", "nyx-executor-pentest:latest"),
		ExecutorWorkspaceRoot:    envOr("NYX_EXECUTOR_WORKSPACE_ROOT", filepath.Join(os.TempDir(), "nyx-executor")),
		ExecutorNetworkMode:      envOr("NYX_EXECUTOR_NETWORK_MODE", "none"),
		ExecutorNetworkName:      envOr("NYX_EXECUTOR_NETWORK_NAME", ""),
		ExecutorEnableNetRaw:     boolOr("NYX_EXECUTOR_ENABLE_NET_RAW", false),
		RequireRiskyApproval:     boolOr("NYX_REQUIRE_RISKY_APPROVAL", true),
		FlowMaxConcurrentActions: intOr("NYX_FLOW_MAX_CONCURRENT_ACTIONS", 1),
		FlowMinActionInterval:    durationOr("NYX_FLOW_MIN_ACTION_INTERVAL", 750*time.Millisecond),
		BrowserMode:              envOr("NYX_BROWSER_MODE", "auto"),
		BrowserTimeout:           durationOr("NYX_BROWSER_TIMEOUT", 20*time.Second),
		BrowserArtifactsRoot:     envOr("NYX_BROWSER_ARTIFACTS_ROOT", filepath.Join(os.TempDir(), "nyx-browser")),
		BrowserHeadless:          boolOr("NYX_BROWSER_HEADLESS", true),
		BrowserExecutablePath:    envOr("NYX_BROWSER_EXECUTABLE", ""),
		SearchMode:               envOr("NYX_SEARCH_MODE", "duckduckgo"),
		SearchWebMode:            envOr("NYX_SEARCH_WEB_MODE", envOr("NYX_SEARCH_MODE", "duckduckgo")),
		SearchDeepMode:           envOr("NYX_SEARCH_DEEP_MODE", "auto"),
		SearchExploitMode:        envOr("NYX_SEARCH_EXPLOIT_MODE", "auto"),
		SearchCodeMode:           envOr("NYX_SEARCH_CODE_MODE", "auto"),
		SearchBaseURL:            envOr("NYX_SEARCH_BASE_URL", ""),
		SearchTimeout:            durationOr("NYX_SEARCH_TIMEOUT", 12*time.Second),
		SearchResultLimit:        intOr("NYX_SEARCH_RESULT_LIMIT", 5),
		SearchUserAgent:          envOr("NYX_SEARCH_USER_AGENT", "NYX/2 autonomous-search"),
		SearchTavilyAPIKey:       envOr("TAVILY_API_KEY", ""),
		SearchTavilyBaseURL:      envOr("NYX_SEARCH_TAVILY_BASE_URL", "https://api.tavily.com/search"),
		SearchPerplexityAPIKey:   envOr("PERPLEXITY_API_KEY", ""),
		SearchPerplexityBaseURL:  envOr("NYX_SEARCH_PERPLEXITY_BASE_URL", "https://api.perplexity.ai/chat/completions"),
		SearchPerplexityModel:    envOr("NYX_SEARCH_PERPLEXITY_MODEL", "sonar-pro"),
		SearchSploitusBaseURL:    envOr("NYX_SEARCH_SPLOITUS_BASE_URL", "https://sploitus.com"),
		APIKey:                   envOr("NYX_API_KEY", ""),
		SupabaseURL:              envOr("SUPABASE_URL", envOr("NEXT_PUBLIC_SUPABASE_URL", "")),
		SupabaseJWTAudience:      envOr("SUPABASE_JWT_AUDIENCE", "authenticated"),
		DefaultTenant:            envOr("NYX_DEFAULT_TENANT", "default"),
		RequireFlowApproval:      boolOr("NYX_REQUIRE_FLOW_APPROVAL", true),
		LogFormat:                envOr("NYX_LOG_FORMAT", "json"),
		LogLevel:                 envOr("NYX_LOG_LEVEL", "info"),
		APIObserveAddr:           envOr("NYX_API_OBSERVE_ADDR", ""),
		OrchestratorObserveAddr:  envOr("NYX_ORCHESTRATOR_OBSERVE_ADDR", ""),
		ExecutorObserveAddr:      envOr("NYX_EXECUTOR_OBSERVE_ADDR", ""),
		HTTPRateLimitRequests:    intOr("NYX_HTTP_RATE_LIMIT_REQUESTS", 120),
		HTTPRateLimitWindow:      durationOr("NYX_HTTP_RATE_LIMIT_WINDOW", time.Minute),
		OpenAIAPIKey:             envOr("OPENAI_API_KEY", ""),
		OpenAIBaseURL:            envOr("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		OpenAIModel:              envOr("OPENAI_MODEL", "gpt-5.1-codex-mini"),
		OpenAIReasoningEffort:    envOr("OPENAI_REASONING_EFFORT", "high"),
		OpenAIMaxOutputTokens:    intOr("OPENAI_MAX_OUTPUT_TOKENS", 8000),
		MemoryEmbeddingsMode:     envOr("NYX_MEMORY_EMBEDDINGS_MODE", "auto"),
		OpenAIEmbeddingModel:     envOr("OPENAI_EMBEDDING_MODEL", "text-embedding-3-small"),
		OpenAIEmbeddingDims:      intOr("OPENAI_EMBEDDING_DIMS", 1536),
		CORSAllowedOrigins:       envOr("NYX_CORS_ALLOWED_ORIGINS", "*"),
		DBMaxOpenConns:           intOr("NYX_DB_MAX_OPEN_CONNS", 10),
		DBMaxIdleConns:           intOr("NYX_DB_MAX_IDLE_CONNS", 5),
		DBConnMaxLifetime:        durationOr("NYX_DB_CONN_MAX_LIFETIME", 30*time.Minute),
		DBConnMaxIdleTime:        durationOr("NYX_DB_CONN_MAX_IDLE_TIME", 5*time.Minute),
	}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func durationOr(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return fallback
}

func intOr(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return fallback
}

func boolOr(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return fallback
}

func (c Config) Validate(service string) error {
	var issues []error

	switch strings.ToLower(strings.TrimSpace(service)) {
	case "api":
		if strings.TrimSpace(c.ListenAddr) == "" {
			issues = append(issues, errors.New("NYX_LISTEN_ADDR is required for api"))
		}
	case "orchestrator", "executor", "migrate":
		if strings.TrimSpace(c.DatabaseURL) == "" {
			issues = append(issues, fmt.Errorf("DATABASE_URL is required for %s", service))
		}
	}

	if mode := strings.ToLower(strings.TrimSpace(c.ExecutorMode)); mode != "auto" && mode != "local" && mode != "docker" {
		issues = append(issues, fmt.Errorf("unsupported NYX_EXECUTOR_MODE %q", c.ExecutorMode))
	}
	if mode := strings.ToLower(strings.TrimSpace(c.ExecutorNetworkMode)); mode != "none" && mode != "bridge" && mode != "custom" {
		issues = append(issues, fmt.Errorf("unsupported NYX_EXECUTOR_NETWORK_MODE %q", c.ExecutorNetworkMode))
	}
	if strings.ToLower(strings.TrimSpace(c.ExecutorNetworkMode)) == "custom" && strings.TrimSpace(c.ExecutorNetworkName) == "" {
		issues = append(issues, errors.New("NYX_EXECUTOR_NETWORK_NAME is required when NYX_EXECUTOR_NETWORK_MODE=custom"))
	}
	if mode := strings.ToLower(strings.TrimSpace(c.AgentRuntimeMode)); mode != "auto" && mode != "deterministic" && mode != "openai" {
		issues = append(issues, fmt.Errorf("unsupported NYX_AGENT_RUNTIME_MODE %q", c.AgentRuntimeMode))
	}
	if mode := strings.ToLower(strings.TrimSpace(c.MemoryEmbeddingsMode)); mode != "auto" && mode != "hash" && mode != "openai" {
		issues = append(issues, fmt.Errorf("unsupported NYX_MEMORY_EMBEDDINGS_MODE %q", c.MemoryEmbeddingsMode))
	}
	if mode := strings.ToLower(strings.TrimSpace(c.BrowserMode)); mode != "auto" && mode != "chromedp" && mode != "http" {
		issues = append(issues, fmt.Errorf("unsupported NYX_BROWSER_MODE %q", c.BrowserMode))
	}
	if mode := strings.ToLower(strings.TrimSpace(c.SearchMode)); !isSupportedWebSearchMode(mode) {
		issues = append(issues, fmt.Errorf("unsupported NYX_SEARCH_MODE %q", c.SearchMode))
	}
	if mode := strings.ToLower(strings.TrimSpace(c.SearchWebMode)); mode != "" && mode != "auto" && !isSupportedWebSearchMode(mode) {
		issues = append(issues, fmt.Errorf("unsupported NYX_SEARCH_WEB_MODE %q", c.SearchWebMode))
	}
	if mode := strings.ToLower(strings.TrimSpace(c.SearchDeepMode)); mode != "" && mode != "auto" && mode != "tavily" && mode != "perplexity" && mode != "searxng" && mode != "disabled" {
		issues = append(issues, fmt.Errorf("unsupported NYX_SEARCH_DEEP_MODE %q", c.SearchDeepMode))
	}
	if mode := strings.ToLower(strings.TrimSpace(c.SearchExploitMode)); mode != "" && mode != "auto" && mode != "sploitus" && mode != "duckduckgo" && mode != "searxng" && mode != "disabled" {
		issues = append(issues, fmt.Errorf("unsupported NYX_SEARCH_EXPLOIT_MODE %q", c.SearchExploitMode))
	}
	if mode := strings.ToLower(strings.TrimSpace(c.SearchCodeMode)); mode != "" && mode != "auto" && mode != "duckduckgo" && mode != "searxng" && mode != "disabled" {
		issues = append(issues, fmt.Errorf("unsupported NYX_SEARCH_CODE_MODE %q", c.SearchCodeMode))
	}
	if format := strings.ToLower(strings.TrimSpace(c.LogFormat)); format != "json" && format != "text" {
		issues = append(issues, fmt.Errorf("unsupported NYX_LOG_FORMAT %q", c.LogFormat))
	}
	if level := strings.ToLower(strings.TrimSpace(c.LogLevel)); level != "debug" && level != "info" && level != "warn" && level != "error" {
		issues = append(issues, fmt.Errorf("unsupported NYX_LOG_LEVEL %q", c.LogLevel))
	}
	if c.QueueMaxDeliver < 1 {
		issues = append(issues, errors.New("NYX_QUEUE_MAX_DELIVER must be >= 1"))
	}
	if c.FlowMaxConcurrentActions < 1 {
		issues = append(issues, errors.New("NYX_FLOW_MAX_CONCURRENT_ACTIONS must be >= 1"))
	}
	if c.FlowMinActionInterval < 0 {
		issues = append(issues, errors.New("NYX_FLOW_MIN_ACTION_INTERVAL must be >= 0"))
	}
	if c.OpenAIMaxOutputTokens < 256 {
		issues = append(issues, errors.New("OPENAI_MAX_OUTPUT_TOKENS must be >= 256"))
	}
	if c.OpenAIEmbeddingDims < 16 {
		issues = append(issues, errors.New("OPENAI_EMBEDDING_DIMS must be >= 16"))
	}
	if c.SearchResultLimit < 1 || c.SearchResultLimit > 10 {
		issues = append(issues, errors.New("NYX_SEARCH_RESULT_LIMIT must be between 1 and 10"))
	}
	if c.SearchTimeout <= 0 {
		issues = append(issues, errors.New("NYX_SEARCH_TIMEOUT must be positive"))
	}
	if c.HTTPRateLimitRequests < 0 {
		issues = append(issues, errors.New("NYX_HTTP_RATE_LIMIT_REQUESTS must be >= 0"))
	}
	if c.HTTPRateLimitWindow < 0 {
		issues = append(issues, errors.New("NYX_HTTP_RATE_LIMIT_WINDOW must be >= 0"))
	}
	if strings.ToLower(strings.TrimSpace(c.AgentRuntimeMode)) == "openai" {
		if strings.TrimSpace(c.OpenAIAPIKey) == "" {
			issues = append(issues, errors.New("OPENAI_API_KEY is required when NYX_AGENT_RUNTIME_MODE=openai"))
		}
		if strings.TrimSpace(c.OpenAIModel) == "" {
			issues = append(issues, errors.New("OPENAI_MODEL is required when NYX_AGENT_RUNTIME_MODE=openai"))
		}
	}
	if strings.TrimSpace(c.OpenAIAPIKey) != "" {
		if strings.TrimSpace(c.OpenAIModel) == "" {
			issues = append(issues, errors.New("OPENAI_MODEL is required when OPENAI_API_KEY is configured"))
		}
		switch effort := strings.ToLower(strings.TrimSpace(c.OpenAIReasoningEffort)); effort {
		case "none", "minimal", "low", "medium", "high", "xhigh":
		default:
			issues = append(issues, fmt.Errorf("unsupported OPENAI_REASONING_EFFORT %q", c.OpenAIReasoningEffort))
		}
		if !strings.HasPrefix(strings.TrimSpace(c.OpenAIBaseURL), "http://") && !strings.HasPrefix(strings.TrimSpace(c.OpenAIBaseURL), "https://") {
			issues = append(issues, fmt.Errorf("OPENAI_BASE_URL must be an absolute http(s) URL"))
		}
	}
	if strings.ToLower(strings.TrimSpace(c.MemoryEmbeddingsMode)) == "openai" && strings.TrimSpace(c.OpenAIAPIKey) == "" {
		issues = append(issues, errors.New("OPENAI_API_KEY is required when NYX_MEMORY_EMBEDDINGS_MODE=openai"))
	}
	if strings.TrimSpace(c.SearchBaseURL) != "" && !strings.HasPrefix(strings.TrimSpace(c.SearchBaseURL), "http://") && !strings.HasPrefix(strings.TrimSpace(c.SearchBaseURL), "https://") {
		issues = append(issues, fmt.Errorf("NYX_SEARCH_BASE_URL must be an absolute http(s) URL"))
	}
	if webSearchUsesProvider(c, "tavily") || deepSearchUsesProvider(c, "tavily") {
		if strings.TrimSpace(c.SearchTavilyAPIKey) == "" {
			issues = append(issues, errors.New("TAVILY_API_KEY is required when a Tavily-backed search mode is enabled"))
		}
		if !isEmptyOrAbsHTTPURL(c.SearchTavilyBaseURL) {
			issues = append(issues, fmt.Errorf("NYX_SEARCH_TAVILY_BASE_URL must be an absolute http(s) URL"))
		}
	}
	if deepSearchUsesProvider(c, "perplexity") {
		if strings.TrimSpace(c.SearchPerplexityAPIKey) == "" {
			issues = append(issues, errors.New("PERPLEXITY_API_KEY is required when NYX_SEARCH_DEEP_MODE=perplexity"))
		}
		if !isEmptyOrAbsHTTPURL(c.SearchPerplexityBaseURL) {
			issues = append(issues, fmt.Errorf("NYX_SEARCH_PERPLEXITY_BASE_URL must be an absolute http(s) URL"))
		}
		if strings.TrimSpace(c.SearchPerplexityModel) == "" {
			issues = append(issues, errors.New("NYX_SEARCH_PERPLEXITY_MODEL is required when NYX_SEARCH_DEEP_MODE=perplexity"))
		}
	}
	if exploitSearchUsesProvider(c, "sploitus") {
		if !isEmptyOrAbsHTTPURL(c.SearchSploitusBaseURL) {
			issues = append(issues, fmt.Errorf("NYX_SEARCH_SPLOITUS_BASE_URL must be an absolute http(s) URL"))
		}
	}
	if strings.TrimSpace(c.ExecutorImage) == "" {
		issues = append(issues, errors.New("NYX_EXECUTOR_IMAGE must not be empty"))
	}
	if strings.TrimSpace(c.SupabaseURL) != "" {
		if !strings.HasPrefix(strings.TrimSpace(c.SupabaseURL), "http://") && !strings.HasPrefix(strings.TrimSpace(c.SupabaseURL), "https://") {
			issues = append(issues, fmt.Errorf("SUPABASE_URL must be an absolute http(s) URL"))
		}
		if strings.TrimSpace(c.SupabaseJWTAudience) == "" {
			issues = append(issues, errors.New("SUPABASE_JWT_AUDIENCE is required when SUPABASE_URL is configured"))
		}
	}

	if c.NATSURL != "" {
		for key, value := range map[string]string{
			"NYX_FLOW_STREAM":           c.FlowStream,
			"NYX_FLOW_SUBJECT":          c.FlowSubject,
			"NYX_FLOW_CONSUMER":         c.FlowConsumer,
			"NYX_ACTION_STREAM":         c.ActionStream,
			"NYX_ACTION_SUBJECT":        c.ActionSubject,
			"NYX_ACTION_CONSUMER":       c.ActionConsumer,
			"NYX_ACTION_RESULT_STREAM":  c.ActionResultStream,
			"NYX_ACTION_RESULT_SUBJECT": c.ActionResultSubject,
			"NYX_EVENT_STREAM":          c.EventStream,
			"NYX_EVENT_SUBJECT":         c.EventSubject,
			"NYX_DLQ_STREAM":            c.DLQStream,
			"NYX_DLQ_SUBJECT":           c.DLQSubject,
		} {
			if strings.TrimSpace(value) == "" {
				issues = append(issues, fmt.Errorf("%s must be set when NATS_URL is configured", key))
			}
		}
	}

	if err := validateOptionalAddr(c.APIObserveAddr, "NYX_API_OBSERVE_ADDR"); err != nil {
		issues = append(issues, err)
	}
	if err := validateOptionalAddr(c.OrchestratorObserveAddr, "NYX_ORCHESTRATOR_OBSERVE_ADDR"); err != nil {
		issues = append(issues, err)
	}
	if err := validateOptionalAddr(c.ExecutorObserveAddr, "NYX_EXECUTOR_OBSERVE_ADDR"); err != nil {
		issues = append(issues, err)
	}

	if len(issues) == 0 {
		return nil
	}
	return errors.Join(issues...)
}

func isSupportedWebSearchMode(mode string) bool {
	switch mode {
	case "duckduckgo", "searxng", "tavily", "perplexity", "disabled":
		return true
	default:
		return false
	}
}

func searchModeRequires(mode, fallback, expected string) bool {
	if mode == expected {
		return true
	}
	if mode == "" || mode == "auto" {
		return fallback == expected
	}
	return false
}

func webSearchUsesProvider(c Config, expected string) bool {
	return searchModeRequires(strings.ToLower(strings.TrimSpace(c.SearchWebMode)), strings.ToLower(strings.TrimSpace(c.SearchMode)), expected)
}

func deepSearchUsesProvider(c Config, expected string) bool {
	mode := strings.ToLower(strings.TrimSpace(c.SearchDeepMode))
	switch mode {
	case expected:
		return true
	case "", "auto":
		if expected == "perplexity" {
			return strings.TrimSpace(c.SearchPerplexityAPIKey) != ""
		}
		if expected == "tavily" {
			return strings.TrimSpace(c.SearchPerplexityAPIKey) == "" &&
				(strings.TrimSpace(c.SearchTavilyAPIKey) != "" || webSearchUsesProvider(c, "tavily"))
		}
		return false
	default:
		return false
	}
}

func exploitSearchUsesProvider(c Config, expected string) bool {
	mode := strings.ToLower(strings.TrimSpace(c.SearchExploitMode))
	if mode == expected {
		return true
	}
	return expected == "sploitus" && (mode == "" || mode == "auto")
}

func isAbsHTTPURL(raw string) bool {
	value := strings.TrimSpace(raw)
	return strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
}

func isEmptyOrAbsHTTPURL(raw string) bool {
	value := strings.TrimSpace(raw)
	return value == "" || isAbsHTTPURL(value)
}

func validateOptionalAddr(addr, key string) error {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return nil
	}
	if _, _, err := net.SplitHostPort(addr); err == nil {
		return nil
	}
	if strings.HasPrefix(addr, ":") {
		if _, err := strconv.Atoi(strings.TrimPrefix(addr, ":")); err == nil {
			return nil
		}
	}
	return fmt.Errorf("%s must be host:port or :port", key)
}
