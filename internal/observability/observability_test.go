package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthReadyHandlerReportsDegradedChecks(t *testing.T) {
	health := Health{
		Service: "api",
		Checks: map[string]CheckFunc{
			"database": func(context.Context) error { return nil },
			"queue": func(context.Context) error {
				return context.DeadlineExceeded
			},
		},
	}

	rec := httptest.NewRecorder()
	health.ReadyHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected degraded status, got %d", rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if payload["status"] != "degraded" {
		t.Fatalf("unexpected health payload: %+v", payload)
	}
}

func TestRegistryRenderSortsAndSanitizesMetrics(t *testing.T) {
	registry := NewRegistry()
	registry.IncCounter("nyx.requests-total", map[string]string{"path": "/api/v1/flows"}, 2)
	registry.SetGauge("nyx.inflight", nil, 1)

	rendered := registry.Render()
	if !strings.Contains(rendered, `nyx_inflight 1`) {
		t.Fatalf("expected inflight gauge, got %q", rendered)
	}
	if !strings.Contains(rendered, `nyx_requests_total{path="/api/v1/flows"} 2`) {
		t.Fatalf("expected sanitized counter key, got %q", rendered)
	}
}

func TestNewLoggerWritesStructuredOutput(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(LoggerConfig{
		Service: "api",
		Format:  "json",
		Level:   "debug",
		Writer:  &buf,
	})
	logger.Info("request complete", "trace_id", "trace-123")

	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("decode log output: %v", err)
	}
	if payload["service"] != "api" {
		t.Fatalf("expected service field, got %+v", payload)
	}
	if payload["trace_id"] != "trace-123" {
		t.Fatalf("expected trace_id field, got %+v", payload)
	}
}

func TestEnsureTracePreservesExistingTrace(t *testing.T) {
	ctx := WithTrace(context.Background(), "trace-1")
	next, traceID := EnsureTrace(ctx)
	if traceID != "trace-1" {
		t.Fatalf("expected trace-1, got %q", traceID)
	}
	if TraceID(next) != "trace-1" {
		t.Fatalf("expected trace in context, got %q", TraceID(next))
	}
}
