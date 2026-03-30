//go:build integration

package integration

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"nyx/internal/executor"
	"nyx/internal/functions"
	"nyx/internal/ids"
	"nyx/internal/services/browser"
	"nyx/internal/services/memory"
	"nyx/internal/store"
)

func TestDockerExecutorCustomNetworkCanReachScopedLocalTarget(t *testing.T) {
	const pentestImage = "nyx-executor-pentest:latest"

	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker CLI is required for docker network integration tests")
	}
	if err := dockerImageAvailable(pentestImage); err != nil {
		t.Skipf("%s is required for docker network integration tests: %v", pentestImage, err)
	}

	networkName := ids.New("nyxnet")
	manager := executor.NewDockerManagerWithConfig(
		"alpine:3.20",
		pentestImage,
		t.TempDir(),
		"custom",
		networkName,
		false,
	)
	if err := manager.Validate(); err != nil {
		t.Skipf("docker executor unavailable: %v", err)
	}
	targetName := ids.New("nyx-target")

	createCtx, cancelCreate := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelCreate()
	if err := dockerCommand(createCtx, "network", "create", networkName).Run(); err != nil {
		t.Fatalf("create docker network: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		_ = dockerCommand(cleanupCtx, "network", "rm", networkName).Run()
	})

	runCtx, cancelRun := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelRun()
	serveScript := "mkdir -p /tmp/serve && printf 'nyx phase2 network ok\\n' > /tmp/serve/index.html && exec python3 -m http.server 8080 -d /tmp/serve"
	if err := dockerCommand(
		runCtx,
		"run",
		"-d",
		"--rm",
		"--network", networkName,
		"--name", targetName,
		pentestImage,
		"sh", "-lc", serveScript,
	).Run(); err != nil {
		t.Fatalf("start target container: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		_ = dockerCommand(cleanupCtx, "rm", "-f", targetName).Run()
	})

	waitCtx, cancelWait := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelWait()
	waitForContainerReadiness(t, waitCtx, pentestImage, networkName, targetName)

	execCtx, cancelExec := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelExec()
	result, err := manager.Execute(execCtx, executor.Request{
		FlowID:       "flow-phase2-network",
		ActionID:     "action-curl-local-target",
		Profile:      "terminal",
		FunctionName: "terminal",
		Input: map[string]string{
			"toolset": "pentest",
			"target":  "http://" + targetName,
			"goal":    "verify scoped network access to a local authorized target",
			"command": fmt.Sprintf("curl -fsS http://%s:8080/", targetName),
		},
	})
	if err != nil {
		t.Fatalf("expected scoped network execution to succeed: %v\nstdout:\n%s\nstderr:\n%s", err, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "nyx phase2 network ok") {
		t.Fatalf("expected local target response in stdout, got stdout=%q stderr=%q", result.Stdout, result.Stderr)
	}
	if result.Metadata["network_mode"] != "custom" {
		t.Fatalf("expected custom network mode metadata, got %q", result.Metadata["network_mode"])
	}
	if result.Metadata["network_name"] != networkName {
		t.Fatalf("expected network name metadata %q, got %q", networkName, result.Metadata["network_name"])
	}
}

func TestGatewayRejectsOutOfScopeNetworkCommandBeforeDockerExecution(t *testing.T) {
	manager := executor.NewDockerManagerWithConfig(
		"alpine:3.20",
		"nyx-executor-pentest:latest",
		t.TempDir(),
		"bridge",
		"",
		false,
	)
	repo := store.NewMemoryStore()
	gateway := functions.NewGateway(browser.NewService(), memory.New(repo), manager)

	result := gateway.Call(context.Background(), "flow-1", "action-1", "terminal", map[string]string{
		"target":  "https://allowed.example.com",
		"toolset": "pentest",
		"command": "curl -fsS https://evil.example.net/",
	})
	if result.Err == nil {
		t.Fatal("expected out-of-scope command to fail")
	}
	if !strings.Contains(result.Output["summary"], "out-of-scope host") {
		t.Fatalf("unexpected summary: %q", result.Output["summary"])
	}
	if result.Output["workspace"] != "" {
		t.Fatalf("expected no workspace metadata when scope validation fails, got %q", result.Output["workspace"])
	}
}

func dockerImageAvailable(image string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return dockerCommand(ctx, "image", "inspect", image).Run()
}

func dockerCommand(ctx context.Context, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, "docker", args...)
}

func waitForContainerReadiness(t *testing.T, ctx context.Context, image, networkName, targetName string) {
	t.Helper()

	command := fmt.Sprintf("curl -fsS http://%s:8080/", targetName)
	for {
		err := dockerCommand(
			ctx,
			"run",
			"--rm",
			"--network", networkName,
			image,
			"sh", "-lc", command,
		).Run()
		if err == nil {
			return
		}
		if ctx.Err() != nil {
			var stderr bytes.Buffer
			cmd := dockerCommand(
				context.Background(),
				"logs",
				targetName,
			)
			cmd.Stderr = &stderr
			cmd.Stdout = &stderr
			_ = cmd.Run()
			t.Fatalf("timed out waiting for local target container readiness: %v\ncontainer logs:\n%s", err, stderr.String())
		}
		time.Sleep(250 * time.Millisecond)
	}
}
