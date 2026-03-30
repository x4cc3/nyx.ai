//go:build integration

package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
	"time"

	"nyx/internal/services/browser"
)

func TestChromedpCapturesRenderedPageAndScreenshot(t *testing.T) {
	if _, err := exec.LookPath("chromium"); err != nil {
		if _, altErr := exec.LookPath("chromium-browser"); altErr != nil {
			t.Skip("chromium or chromium-browser is required for browser integration tests")
		}
	}

	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html>
<html>
  <head><title>NYX Browser IT</title></head>
  <body>
    <main id="status">booting</main>
    <script>
      setTimeout(function () {
        var ready = document.createElement("div");
        ready.id = "ready";
        ready.textContent = "phase5 rendered content";
        document.body.appendChild(ready);
      }, 50);
    </script>
  </body>
</html>`))
	}))
	defer target.Close()

	service := browser.NewServiceWithRuntime(browser.RuntimeConfig{
		Mode:          "chromedp",
		Timeout:       10 * time.Second,
		ArtifactsRoot: t.TempDir(),
		Headless:      true,
	})

	result := service.Navigate(context.Background(), browser.Request{
		URL:          target.URL,
		WaitSelector: "#ready",
	})

	if result.Mode != "chromedp" {
		t.Fatalf("expected chromedp mode, got %q with summary %q", result.Mode, result.Summary)
	}
	if result.ScreenshotPath == "" {
		t.Fatal("expected screenshot path to be populated")
	}
	if result.SnapshotPath == "" {
		t.Fatal("expected html snapshot path to be populated")
	}
	if !strings.Contains(result.Text, "phase5 rendered content") {
		t.Fatalf("expected rendered JavaScript content in text capture, got %q", result.Text)
	}
}
