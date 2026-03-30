package functions

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"nyx/internal/domain"
	"nyx/internal/executor"
	"nyx/internal/services/browser"
	"nyx/internal/services/memory"
	"nyx/internal/services/search"
	"nyx/internal/store"
)

func TestTerminalCallsUseExecutor(t *testing.T) {
	repo := store.NewMemoryStore()
	mem := memory.New(repo)
	gateway := NewGateway(browser.NewService(), mem, executor.NewLocalManagerWithRoot(t.TempDir()))

	result := gateway.Call(context.Background(), "flow-1", "action-1", "terminal", map[string]string{
		"goal": "capture baseline terminal telemetry",
	})
	if result.Err != nil {
		t.Fatalf("expected nil error, got %v", result.Err)
	}
	if result.Runtime == "" {
		t.Fatal("expected executor runtime to be set")
	}
	if result.Output["summary"] == "" {
		t.Fatal("expected terminal summary to be set")
	}
	if result.Output["workspace"] == "" {
		t.Fatal("expected terminal workspace metadata to be set")
	}
	if result.Output["exit_code"] != "0" {
		t.Fatalf("expected exit code 0, got %s", result.Output["exit_code"])
	}
}

func TestSearchWebUsesConfiguredSearchService(t *testing.T) {
	repo := store.NewMemoryStore()
	mem := memory.New(repo)
	searchService := search.NewServiceWithConfig(search.Config{
		Mode: "duckduckgo",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				body := `<html><body><a class="result__a" href="https://docs.example.com">Acme Docs</a></body></html>`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(body)),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			}),
		},
	})
	gateway := NewGateway(
		browser.NewService(),
		mem,
		executor.NewLocalManagerWithRoot(t.TempDir()),
		WithSearchService(searchService),
	)

	result := gateway.Call(context.Background(), "flow-1", "action-1", "search_web", map[string]string{
		"query": "acme",
	})
	if result.Err != nil {
		t.Fatalf("expected nil error, got %v", result.Err)
	}
	if result.Runtime != "go-search-service" {
		t.Fatalf("unexpected runtime: %s", result.Runtime)
	}
	if result.Output["source"] != "duckduckgo-html" {
		t.Fatalf("unexpected source: %s", result.Output["source"])
	}
	if result.Output["result_1_title"] != "Acme Docs" {
		t.Fatalf("unexpected title: %s", result.Output["result_1_title"])
	}
}

func TestDefinitionsExposeRegistryMetadata(t *testing.T) {
	repo := store.NewMemoryStore()
	mem := memory.New(repo)
	gateway := NewGateway(browser.NewService(), mem, executor.NewLocalManagerWithRoot(t.TempDir()))

	defs := gateway.Definitions()
	if len(defs) <= 7 {
		t.Fatalf("expected expanded registry, got %d definitions", len(defs))
	}

	var terminalExec domain.FunctionDef
	found := false
	for _, def := range defs {
		if def.Name == "terminal_exec" {
			terminalExec = def
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected terminal_exec definition")
	}
	if terminalExec.Category != "environment" {
		t.Fatalf("unexpected category: %q", terminalExec.Category)
	}
	if !terminalExec.RequiresPentestImage {
		t.Fatal("expected terminal_exec to require the pentest image")
	}
	if len(terminalExec.InputSchema) == 0 {
		t.Fatal("expected input schema metadata")
	}
}

func TestTerminalExecDefaultsToPentestToolset(t *testing.T) {
	repo := store.NewMemoryStore()
	mem := memory.New(repo)
	gateway := NewGateway(browser.NewService(), mem, executor.NewLocalManagerWithRoot(t.TempDir()))

	result := gateway.Call(context.Background(), "flow-1", "action-1", "terminal_exec", map[string]string{
		"target":  "https://app.example.com",
		"command": "printf ok",
	})
	if result.Err != nil {
		t.Fatalf("expected nil error, got %v", result.Err)
	}
	if result.Output["toolset"] != "pentest" {
		t.Fatalf("expected pentest toolset, got %q", result.Output["toolset"])
	}
}

func TestExtractLinksNormalizesAndDeduplicatesLinks(t *testing.T) {
	links := extractLinks(
		`<html><body><a href="/docs">Docs</a><a href="https://example.com/login">Login</a><a href="/docs">Docs again</a></body></html>`,
		"https://app.example.com/start",
	)
	if len(links) != 2 {
		t.Fatalf("expected 2 normalized links, got %+v", links)
	}
	if links[0] != "https://app.example.com/docs" {
		t.Fatalf("unexpected first link: %q", links[0])
	}
	if links[1] != "https://example.com/login" {
		t.Fatalf("unexpected second link: %q", links[1])
	}
}

func TestSearchExploitsExpandsTheQuery(t *testing.T) {
	repo := store.NewMemoryStore()
	mem := memory.New(repo)
	var capturedQuery string
	searchService := search.NewServiceWithConfig(search.Config{
		Mode: "duckduckgo",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				capturedQuery = req.URL.Query().Get("q")
				body := `<html><body><a class="result__a" href="https://docs.example.com/poc">PoC</a></body></html>`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(body)),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			}),
		},
	})
	gateway := NewGateway(
		browser.NewService(),
		mem,
		executor.NewLocalManagerWithRoot(t.TempDir()),
		WithSearchService(searchService),
	)

	result := gateway.Call(context.Background(), "flow-1", "action-1", "search_exploits", map[string]string{
		"target":    "app.example.com",
		"objective": "investigate authentication bypass",
	})
	if result.Err != nil {
		t.Fatalf("expected nil error, got %v", result.Err)
	}
	if !strings.Contains(strings.ToLower(capturedQuery), "exploit") {
		t.Fatalf("expected exploit-oriented query, got %q", capturedQuery)
	}
}

func TestSearchCodeUsesCodeKindQueryExpansion(t *testing.T) {
	repo := store.NewMemoryStore()
	mem := memory.New(repo)
	var capturedQuery string
	searchService := search.NewServiceWithConfig(search.Config{
		Mode: "duckduckgo",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				capturedQuery = req.URL.Query().Get("q")
				body := `<html><body><a class="result__a" href="https://docs.example.com/reference">Reference</a></body></html>`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(body)),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			}),
		},
	})
	gateway := NewGateway(
		browser.NewService(),
		mem,
		executor.NewLocalManagerWithRoot(t.TempDir()),
		WithSearchService(searchService),
	)

	result := gateway.Call(context.Background(), "flow-1", "action-1", "search_code", map[string]string{
		"target":    "app.example.com",
		"objective": "identify the API reference path",
	})
	if result.Err != nil {
		t.Fatalf("expected nil error, got %v", result.Err)
	}
	if !strings.Contains(strings.ToLower(capturedQuery), "api reference") {
		t.Fatalf("expected code/manual query expansion, got %q", capturedQuery)
	}
}

func TestSearchMemoryCanFilterByNamespace(t *testing.T) {
	repo := store.NewMemoryStore()
	mem := memory.New(repo)
	if _, err := mem.StoreTargetObservation(context.Background(), "flow-1", "action-1", "Captured login page and session bootstrap.", nil); err != nil {
		t.Fatalf("store target observation: %v", err)
	}
	if _, err := mem.StoreExploitReference(context.Background(), "flow-1", "action-2", "Exploit research: auth bypass PoC.", nil); err != nil {
		t.Fatalf("store exploit reference: %v", err)
	}
	gateway := NewGateway(browser.NewService(), mem, executor.NewLocalManagerWithRoot(t.TempDir()))

	result := gateway.Call(context.Background(), "flow-1", "action-3", "search_memory", map[string]string{
		"query":     "auth bypass",
		"namespace": domain.MemoryNamespaceExploitReferences,
	})
	if result.Err != nil {
		t.Fatalf("expected nil error, got %v", result.Err)
	}
	if result.Output["matches"] != "1" {
		t.Fatalf("expected one namespaced memory match, got %q", result.Output["matches"])
	}
	if result.Output["namespace"] != domain.MemoryNamespaceExploitReferences {
		t.Fatalf("expected exploit namespace output, got %q", result.Output["namespace"])
	}
	if !strings.Contains(result.Output["summary"], domain.MemoryNamespaceExploitReferences) {
		t.Fatalf("expected namespaced summary, got %q", result.Output["summary"])
	}
}

func TestBrowserRejectsOutOfScopeNavigation(t *testing.T) {
	repo := store.NewMemoryStore()
	mem := memory.New(repo)
	gateway := NewGateway(browser.NewService(), mem, executor.NewLocalManagerWithRoot(t.TempDir()))

	result := gateway.Call(context.Background(), "flow-1", "action-1", "browser", map[string]string{
		"target": "https://app.example.com",
		"url":    "https://evil.example.net",
	})
	if result.Err == nil {
		t.Fatal("expected out-of-scope browser navigation to fail")
	}
	if !strings.Contains(result.Output["summary"], "outside the allowed scope") {
		t.Fatalf("unexpected summary: %q", result.Output["summary"])
	}
}

func TestTerminalRejectsOutOfScopeCommand(t *testing.T) {
	repo := store.NewMemoryStore()
	mem := memory.New(repo)
	gateway := NewGateway(browser.NewService(), mem, executor.NewLocalManagerWithRoot(t.TempDir()))

	result := gateway.Call(context.Background(), "flow-1", "action-1", "terminal", map[string]string{
		"target":  "https://app.example.com",
		"command": "curl https://evil.example.net",
	})
	if result.Err == nil {
		t.Fatal("expected out-of-scope terminal command to fail")
	}
	if !strings.Contains(result.Output["summary"], "out-of-scope host") {
		t.Fatalf("unexpected summary: %q", result.Output["summary"])
	}
}

func TestTerminalRequiresTargetWhenExecutingCommand(t *testing.T) {
	repo := store.NewMemoryStore()
	mem := memory.New(repo)
	gateway := NewGateway(browser.NewService(), mem, executor.NewLocalManagerWithRoot(t.TempDir()))

	result := gateway.Call(context.Background(), "flow-1", "action-1", "terminal", map[string]string{
		"command": "curl https://app.example.com",
	})
	if result.Err == nil {
		t.Fatal("expected terminal command without target to fail")
	}
	if !strings.Contains(result.Output["summary"], "require an in-scope target") {
		t.Fatalf("unexpected summary: %q", result.Output["summary"])
	}
}

func TestBrowserAllowsSiblingHostWithinRegistrableDomain(t *testing.T) {
	repo := store.NewMemoryStore()
	mem := memory.New(repo)
	gateway := NewGateway(browser.NewService(), mem, executor.NewLocalManagerWithRoot(t.TempDir()))

	result := gateway.Call(context.Background(), "flow-1", "action-1", "browser", map[string]string{
		"target": "https://app.example.com",
		"url":    "https://api.example.com",
	})
	if result.Err != nil {
		t.Fatalf("expected sibling host to be allowed, got %v", result.Err)
	}
}

func TestBrowserRequiresTargetWhenNavigating(t *testing.T) {
	repo := store.NewMemoryStore()
	mem := memory.New(repo)
	gateway := NewGateway(browser.NewService(), mem, executor.NewLocalManagerWithRoot(t.TempDir()))

	result := gateway.Call(context.Background(), "flow-1", "action-1", "browser", map[string]string{
		"url": "https://app.example.com",
	})
	if result.Err == nil {
		t.Fatal("expected browser action without target to fail")
	}
	if !strings.Contains(result.Output["summary"], "require an in-scope target") {
		t.Fatalf("unexpected summary: %q", result.Output["summary"])
	}
}

func TestTerminalAllowsSiblingHostWithinRegistrableDomain(t *testing.T) {
	repo := store.NewMemoryStore()
	mem := memory.New(repo)
	gateway := NewGateway(browser.NewService(), mem, executor.NewLocalManagerWithRoot(t.TempDir()))

	result := gateway.Call(context.Background(), "flow-1", "action-1", "terminal", map[string]string{
		"target":  "https://app.example.co.uk",
		"command": "printf https://cdn.example.co.uk",
	})
	if result.Err != nil {
		t.Fatalf("expected sibling host to be allowed, got %v", result.Err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
