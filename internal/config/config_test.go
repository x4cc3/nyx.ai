package config

import (
	"strings"
	"testing"
)

func TestValidateRejectsUnsupportedModes(t *testing.T) {
	cfg := Config{
		ListenAddr:               ":8080",
		AgentRuntimeMode:         "oracle",
		MemoryEmbeddingsMode:     "void",
		ExecutorMode:             "weird",
		BrowserMode:              "robot",
		SearchMode:               "magic",
		LogFormat:                "yaml",
		LogLevel:                 "verbose",
		QueueMaxDeliver:          0,
		OpenAIMaxOutputTokens:    0,
		OpenAIEmbeddingDims:      0,
		FlowMaxConcurrentActions: 0,
	}
	if err := cfg.Validate("api"); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateAcceptsReasonableAPIConfig(t *testing.T) {
	cfg := Config{
		ListenAddr:               ":8080",
		AgentRuntimeMode:         "auto",
		MemoryEmbeddingsMode:     "auto",
		ExecutorMode:             "auto",
		ExecutorImage:            "alpine:3.20",
		ExecutorNetworkMode:      "none",
		BrowserMode:              "auto",
		SearchMode:               "duckduckgo",
		SearchTimeout:            5,
		SearchResultLimit:        5,
		LogFormat:                "json",
		LogLevel:                 "info",
		QueueMaxDeliver:          3,
		APIObserveAddr:           ":9080",
		OpenAIMaxOutputTokens:    2048,
		OpenAIEmbeddingDims:      1536,
		FlowMaxConcurrentActions: 1,
	}
	if err := cfg.Validate("api"); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}
}

func TestValidateRequiresOpenAIKeyInOpenAIMode(t *testing.T) {
	cfg := Config{
		ListenAddr:               ":8080",
		AgentRuntimeMode:         "openai",
		MemoryEmbeddingsMode:     "auto",
		ExecutorMode:             "auto",
		ExecutorImage:            "alpine:3.20",
		ExecutorNetworkMode:      "none",
		BrowserMode:              "auto",
		SearchMode:               "duckduckgo",
		SearchTimeout:            5,
		SearchResultLimit:        5,
		LogFormat:                "json",
		LogLevel:                 "info",
		QueueMaxDeliver:          3,
		OpenAIModel:              "gpt-5.1-codex-mini",
		OpenAIReasoningEffort:    "high",
		OpenAIMaxOutputTokens:    2048,
		OpenAIEmbeddingDims:      1536,
		FlowMaxConcurrentActions: 1,
	}
	if err := cfg.Validate("orchestrator"); err == nil {
		t.Fatal("expected openai validation error")
	}
}

func TestValidateAcceptsSupabaseConfig(t *testing.T) {
	cfg := Config{
		ListenAddr:               ":8080",
		AgentRuntimeMode:         "auto",
		MemoryEmbeddingsMode:     "auto",
		ExecutorMode:             "auto",
		ExecutorImage:            "alpine:3.20",
		ExecutorNetworkMode:      "none",
		BrowserMode:              "auto",
		SearchMode:               "duckduckgo",
		SearchTimeout:            5,
		SearchResultLimit:        5,
		LogFormat:                "json",
		LogLevel:                 "info",
		QueueMaxDeliver:          3,
		OpenAIMaxOutputTokens:    2048,
		OpenAIEmbeddingDims:      1536,
		FlowMaxConcurrentActions: 1,
		SupabaseURL:              "https://demo.supabase.co",
		SupabaseJWTAudience:      "authenticated",
	}
	if err := cfg.Validate("api"); err != nil {
		t.Fatalf("expected valid supabase config, got %v", err)
	}
}

func TestValidateRequiresCustomExecutorNetworkName(t *testing.T) {
	cfg := Config{
		ListenAddr:               ":8080",
		AgentRuntimeMode:         "auto",
		MemoryEmbeddingsMode:     "auto",
		ExecutorMode:             "auto",
		ExecutorImage:            "alpine:3.20",
		ExecutorNetworkMode:      "custom",
		BrowserMode:              "auto",
		SearchMode:               "duckduckgo",
		SearchTimeout:            5,
		SearchResultLimit:        5,
		LogFormat:                "json",
		LogLevel:                 "info",
		QueueMaxDeliver:          3,
		OpenAIMaxOutputTokens:    2048,
		OpenAIEmbeddingDims:      1536,
		FlowMaxConcurrentActions: 1,
	}
	if err := cfg.Validate("api"); err == nil {
		t.Fatal("expected custom network validation error")
	}
}

func TestValidateRequiresPerplexityConfigWhenDeepSearchAutoSelectsIt(t *testing.T) {
	cfg := Config{
		ListenAddr:               ":8080",
		AgentRuntimeMode:         "auto",
		MemoryEmbeddingsMode:     "auto",
		ExecutorMode:             "auto",
		ExecutorImage:            "alpine:3.20",
		ExecutorNetworkMode:      "none",
		BrowserMode:              "auto",
		SearchMode:               "duckduckgo",
		SearchDeepMode:           "auto",
		SearchTimeout:            5,
		SearchResultLimit:        5,
		SearchPerplexityAPIKey:   "perplexity-key",
		SearchPerplexityBaseURL:  "not-a-url",
		SearchPerplexityModel:    "sonar-pro",
		LogFormat:                "json",
		LogLevel:                 "info",
		QueueMaxDeliver:          3,
		OpenAIMaxOutputTokens:    2048,
		OpenAIEmbeddingDims:      1536,
		FlowMaxConcurrentActions: 1,
	}
	err := cfg.Validate("api")
	if err == nil || !strings.Contains(err.Error(), "NYX_SEARCH_PERPLEXITY_BASE_URL") {
		t.Fatalf("expected perplexity validation error, got %v", err)
	}
}

func TestValidateRequiresSploitusURLWhenExploitSearchDefaultsToIt(t *testing.T) {
	cfg := Config{
		ListenAddr:               ":8080",
		AgentRuntimeMode:         "auto",
		MemoryEmbeddingsMode:     "auto",
		ExecutorMode:             "auto",
		ExecutorImage:            "alpine:3.20",
		ExecutorNetworkMode:      "none",
		BrowserMode:              "auto",
		SearchMode:               "duckduckgo",
		SearchExploitMode:        "auto",
		SearchSploitusBaseURL:    "bad-url",
		SearchTimeout:            5,
		SearchResultLimit:        5,
		LogFormat:                "json",
		LogLevel:                 "info",
		QueueMaxDeliver:          3,
		OpenAIMaxOutputTokens:    2048,
		OpenAIEmbeddingDims:      1536,
		FlowMaxConcurrentActions: 1,
	}
	err := cfg.Validate("api")
	if err == nil || !strings.Contains(err.Error(), "NYX_SEARCH_SPLOITUS_BASE_URL") {
		t.Fatalf("expected sploitus validation error, got %v", err)
	}
}
