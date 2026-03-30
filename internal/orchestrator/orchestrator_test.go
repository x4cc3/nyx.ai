package orchestrator

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"nyx/internal/domain"
	"nyx/internal/executor"
	"nyx/internal/functions"
	"nyx/internal/queue"
	"nyx/internal/services/browser"
	"nyx/internal/services/memory"
	"nyx/internal/services/search"
	"nyx/internal/store"
)

func TestExecuteFunctionPersistsExecutionMetadataAndTraceArtifacts(t *testing.T) {
	repo := store.NewMemoryStore()
	flow, err := repo.CreateFlow(context.Background(), domain.CreateFlowInput{
		Name:      "Phase 8 flow",
		Target:    "local-target",
		Objective: "Capture operator-visible execution traces",
	})
	if err != nil {
		t.Fatalf("create flow: %v", err)
	}

	gateway := functions.NewGateway(
		browser.NewService(),
		memory.New(repo),
		executor.NewLocalManagerWithRoot(t.TempDir()),
	)
	engine := New(repo, gateway, time.Millisecond, time.Second, queue.NewNoopTransport(), nil, nil, false)

	result := engine.executeFunction(context.Background(), flow.ID, "", "", "operator", "terminal_exec", map[string]string{
		"command": "printf 'phase8-stdout'",
		"goal":    "record execution trace",
		"target":  "https://app.example.com",
		"toolset": "pentest",
	})
	if result.Err != nil {
		t.Fatalf("execute function: %v", result.Err)
	}

	detail, err := repo.FlowDetail(context.Background(), flow.ID)
	if err != nil {
		t.Fatalf("flow detail: %v", err)
	}
	if len(detail.Executions) != 1 {
		t.Fatalf("expected 1 execution, got %d", len(detail.Executions))
	}

	exec := detail.Executions[0]
	if exec.Metadata["command"] != "printf 'phase8-stdout'" {
		t.Fatalf("expected command metadata, got %+v", exec.Metadata)
	}
	if exec.Metadata["toolset"] != "pentest" {
		t.Fatalf("expected toolset metadata, got %+v", exec.Metadata)
	}
	if strings.TrimSpace(exec.Metadata["workspace"]) == "" {
		t.Fatalf("expected workspace metadata, got %+v", exec.Metadata)
	}
	if !strings.Contains(exec.Runtime, "local") {
		t.Fatalf("expected local runtime, got %q", exec.Runtime)
	}

	var stdoutArtifactFound bool
	var executionArtifactFound bool
	for _, artifact := range detail.Artifacts {
		switch artifact.Kind {
		case "stdout":
			stdoutArtifactFound = strings.Contains(artifact.Content, "phase8-stdout")
		case "execution":
			executionArtifactFound = strings.Contains(artifact.Content, "command: printf 'phase8-stdout'") &&
				strings.Contains(artifact.Content, "workspace:")
		}
	}
	if !stdoutArtifactFound {
		t.Fatal("expected stdout artifact to be persisted")
	}
	if !executionArtifactFound {
		t.Fatal("expected execution trace artifact to be persisted")
	}
}

func TestExecuteFunctionStoresExploitResearchInMemory(t *testing.T) {
	repo := store.NewMemoryStore()
	flow, err := repo.CreateFlow(context.Background(), domain.CreateFlowInput{
		Name:      "Phase 9 flow",
		Target:    "https://app.example.com",
		Objective: "Capture exploit research in memory",
	})
	if err != nil {
		t.Fatalf("create flow: %v", err)
	}

	searchService := search.NewServiceWithConfig(search.Config{
		Mode: "duckduckgo",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				body := `<html><body><a class="result__a" href="https://example.com/poc">PoC Example</a></body></html>`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(body)),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			}),
		},
	})
	gateway := functions.NewGateway(
		browser.NewService(),
		memory.New(repo),
		executor.NewLocalManagerWithRoot(t.TempDir()),
		functions.WithSearchService(searchService),
	)
	engine := New(repo, gateway, time.Millisecond, time.Second, queue.NewNoopTransport(), nil, nil, false)

	result := engine.executeFunction(context.Background(), flow.ID, "", "", "researcher", "search_exploits", map[string]string{
		"target":    "https://app.example.com",
		"objective": "investigate authentication bypass",
	})
	if result.Err != nil {
		t.Fatalf("execute exploit search: %v", result.Err)
	}

	detail, err := repo.FlowDetail(context.Background(), flow.ID)
	if err != nil {
		t.Fatalf("flow detail: %v", err)
	}
	if len(detail.Memories) == 0 {
		t.Fatal("expected exploit research memory to be stored")
	}
	found := false
	for _, item := range detail.Memories {
		if item.Metadata["namespace"] == domain.MemoryNamespaceExploitReferences {
			found = true
			if !strings.Contains(item.Content, "PoC Example") {
				t.Fatalf("expected stored exploit reference summary, got %q", item.Content)
			}
		}
	}
	if !found {
		t.Fatal("expected exploit namespace memory entry")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
