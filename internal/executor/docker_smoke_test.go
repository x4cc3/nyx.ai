package executor

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestDockerManagerPentestImageSmoke(t *testing.T) {
	if os.Getenv("NYX_RUN_DOCKER_SMOKE") != "1" {
		t.Skip("set NYX_RUN_DOCKER_SMOKE=1 to run docker executor smoke tests")
	}

	manager := NewDockerManagerWithRoot("alpine:3.20", "nyx-executor-pentest:latest", t.TempDir())
	if err := manager.Validate(); err != nil {
		t.Skipf("docker runtime unavailable: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := manager.Execute(ctx, Request{
		FlowID:       "flow-smoke",
		ActionID:     "action-subfinder-httpx",
		Profile:      "terminal",
		FunctionName: "terminal",
		Input: map[string]string{
			"toolset": "pentest",
			"goal":    "verify pentest worker image tools are available",
			"command": "mkdir -p /tmp/.config/subfinder && : > /tmp/.config/subfinder/config.yaml && : > /tmp/.config/subfinder/provider-config.yaml && HOME=/tmp XDG_CONFIG_HOME=/tmp/.config subfinder -version && httpx -version",
		},
	})
	if err != nil {
		t.Fatalf("docker pentest smoke test failed: %v\nstdout:\n%s\nstderr:\n%s", err, result.Stdout, result.Stderr)
	}
	if result.Metadata["toolset"] != toolsetPentest {
		t.Fatalf("expected pentest toolset metadata, got %s", result.Metadata["toolset"])
	}
	if result.Metadata["image"] != "nyx-executor-pentest:latest" {
		t.Fatalf("expected pentest image metadata, got %s", result.Metadata["image"])
	}
	output := result.Stdout + result.Stderr
	if !strings.Contains(output, "Current Version:") {
		t.Fatalf("expected version output from pentest tools, got stdout=%q stderr=%q", result.Stdout, result.Stderr)
	}
}
