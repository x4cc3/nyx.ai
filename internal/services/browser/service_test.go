package browser

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNavigateHTTPFallbackCapturesSnapshotAndAuth(t *testing.T) {
	service := NewServiceWithRuntime(RuntimeConfig{
		Mode:          "http",
		Timeout:       2 * time.Second,
		ArtifactsRoot: t.TempDir(),
	})
	service.client = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
				return &http.Response{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader("missing auth")),
					Header:     make(http.Header),
					Request:    r,
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`<html><head><title>Authenticated</title></head><body><main>runtime browser capture</main></body></html>`,
				)),
				Header:  make(http.Header),
				Request: r,
			}, nil
		}),
	}
	result := service.Navigate(context.Background(), Request{
		URL:        "https://app.example.test",
		AuthHeader: "Bearer test-token",
	})

	if result.Mode != "http" {
		t.Fatalf("expected http mode, got %s", result.Mode)
	}
	if result.Title != "Authenticated" {
		t.Fatalf("expected title to be captured, got %q", result.Title)
	}
	if result.AuthState != "session-replayed" {
		t.Fatalf("expected auth replay state, got %s", result.AuthState)
	}
	if result.SnapshotPath == "" {
		t.Fatal("expected snapshot path to be set")
	}
	if _, err := os.Stat(result.SnapshotPath); err != nil {
		t.Fatalf("expected snapshot file to exist: %v", err)
	}
	if !strings.Contains(result.Summary, "Authenticated") {
		t.Fatalf("expected summary to mention captured title, got %q", result.Summary)
	}
}

func TestNavigateHTTPScreenshotOnlyReportsDegradedCapture(t *testing.T) {
	service := NewServiceWithRuntime(RuntimeConfig{
		Mode:          "http",
		Timeout:       2 * time.Second,
		ArtifactsRoot: t.TempDir(),
	})
	service.client = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`<html><head><title>Rendered</title></head><body><main>ignored html body</main></body></html>`,
				)),
				Header:  make(http.Header),
				Request: r,
			}, nil
		}),
	}

	result := service.Navigate(context.Background(), Request{
		URL:         "https://app.example.test",
		CaptureMode: "screenshot",
	})

	if result.Mode != "http" {
		t.Fatalf("expected http mode, got %s", result.Mode)
	}
	if result.ScreenshotPath != "" {
		t.Fatalf("expected no screenshot path in http mode, got %q", result.ScreenshotPath)
	}
	if result.SnapshotPath != "" {
		t.Fatalf("expected no html snapshot in screenshot-only mode, got %q", result.SnapshotPath)
	}
	if result.Text != "" || result.HTML != "" {
		t.Fatalf("expected screenshot-only response to omit text/html, got text=%q html=%q", result.Text, result.HTML)
	}
	if !strings.Contains(strings.ToLower(result.Summary), "screenshot-only capture is degraded") {
		t.Fatalf("expected degraded screenshot summary, got %q", result.Summary)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
