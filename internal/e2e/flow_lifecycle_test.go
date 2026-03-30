package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"nyx/internal/config"
	"nyx/internal/domain"
	"nyx/internal/executor"
	"nyx/internal/functions"
	"nyx/internal/httpapi"
	"nyx/internal/orchestrator"
	"nyx/internal/queue"
	"nyx/internal/services/browser"
	"nyx/internal/services/memory"
	"nyx/internal/services/search"
	"nyx/internal/store"
)

func TestEndToEndFlowLifecycleAndStreaming(t *testing.T) {
	repo := store.NewMemoryStore()
	memoryService := memory.New(repo)
	browserService := browser.NewServiceWithRuntime(browser.RuntimeConfig{
		Mode:          "http",
		ArtifactsRoot: t.TempDir(),
	})
	searchService := search.NewServiceWithConfig(search.Config{
		Mode: "duckduckgo",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				body := `<html><body><a class="result__a" href="https://docs.example.com">Acme Surface</a></body></html>`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(body)),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			}),
		},
	})
	gateway := functions.NewGateway(browserService, memoryService, executor.NewLocalManagerWithRoot(t.TempDir()), functions.WithSearchService(searchService))
	cfg := config.Config{
		ListenAddr:          ":0",
		ServiceName:         "test-api",
		ExecutorMode:        "local",
		DefaultTenant:       "default",
		RequireFlowApproval: false,
	}
	server := httpapi.NewServer(cfg, repo, gateway, queue.NewNoopTransport(), nil, nil)
	engine := orchestrator.New(repo, gateway, 5*time.Millisecond, time.Second, queue.NewNoopTransport(), nil, nil, false)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = engine.RunForever(ctx) }()

	handler := server.Handler()

	createBody, _ := json.Marshal(domain.CreateFlowInput{
		Name:      "E2E flow",
		Target:    "https://acme-target.example.com",
		Objective: "Exercise the in-memory end-to-end runtime",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/flows", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected create 201, got %d", rec.Code)
	}

	var flow domain.Flow
	if err := json.Unmarshal(rec.Body.Bytes(), &flow); err != nil {
		t.Fatalf("decode created flow: %v", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/flows/"+flow.ID+"/start", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected start 202, got %d", rec.Code)
	}

	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		current, err := repo.GetFlow(context.Background(), flow.ID)
		if err == nil && current.Status == domain.StatusCompleted {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	current, err := repo.GetFlow(context.Background(), flow.ID)
	if err != nil {
		t.Fatalf("get flow: %v", err)
	}
	if current.Status != domain.StatusCompleted {
		t.Fatalf("expected completed flow, got %s", current.Status)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/flows/"+flow.ID, nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected detail 200, got %d", rec.Code)
	}
	var detail domain.FlowDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	if len(detail.Actions) == 0 || len(detail.Tasks) == 0 {
		t.Fatalf("expected runtime artifacts in detail, got %+v", detail)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/flows/"+flow.ID+"/report?format=markdown", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected markdown report 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "# E2E flow Report") {
		t.Fatalf("expected markdown report body, got %q", rec.Body.String())
	}

	streamCtx, streamCancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
	defer streamCancel()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/flows/"+flow.ID+"/events", nil).WithContext(streamCtx)
	sseRec := newFlushRecorder()
	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(sseRec, req)
		close(done)
	}()
	<-done
	body := sseRec.Body.String()
	if !strings.Contains(body, "event: snapshot") {
		t.Fatalf("expected snapshot event, got %q", body)
	}
	if !strings.Contains(body, "event: flow.status") {
		t.Fatalf("expected flow status events, got %q", body)
	}
}

type flushRecorder struct {
	HeaderMap http.Header
	Body      bytes.Buffer
	Code      int
}

func newFlushRecorder() *flushRecorder {
	return &flushRecorder{
		HeaderMap: make(http.Header),
		Code:      http.StatusOK,
	}
}

func (r *flushRecorder) Header() http.Header {
	return r.HeaderMap
}

func (r *flushRecorder) Write(p []byte) (int, error) {
	return r.Body.Write(p)
}

func (r *flushRecorder) WriteHeader(status int) {
	r.Code = status
}

func (r *flushRecorder) Flush() {}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
