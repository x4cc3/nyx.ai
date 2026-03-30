package executor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLocalManagerExecute(t *testing.T) {
	manager := NewLocalManagerWithRoot(t.TempDir())
	result, err := manager.Execute(context.Background(), Request{
		FlowID:       "flow-1",
		ActionID:     "action-1",
		Profile:      "terminal",
		FunctionName: "terminal",
		Input: map[string]string{
			"goal": "validate initial target access",
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if manager.Mode() != "local" {
		t.Fatalf("expected local mode, got %s", manager.Mode())
	}
	if !strings.Contains(result.Summary, "validate initial target access") {
		t.Fatalf("unexpected summary: %s", result.Summary)
	}
	if result.Workspace == "" {
		t.Fatal("expected workspace path to be set")
	}
	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", result.ExitCode)
	}
	if result.Metadata["toolset"] != toolsetGeneral {
		t.Fatalf("expected general toolset metadata, got %s", result.Metadata["toolset"])
	}
}

func TestLocalManagerCreatesIsolatedFileWorkspace(t *testing.T) {
	root := t.TempDir()
	manager := NewLocalManagerWithRoot(root)
	result, err := manager.Execute(context.Background(), Request{
		FlowID:       "flow-alpha",
		ActionID:     "action-file",
		Profile:      "file",
		FunctionName: "file",
		Input: map[string]string{
			"operation": "write",
			"path":      "notes/result.txt",
			"content":   "workspace artifact",
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	writtenFile := filepath.Join(result.Workspace, "notes", "result.txt")
	content, readErr := os.ReadFile(writtenFile)
	if readErr != nil {
		t.Fatalf("read written file: %v", readErr)
	}
	if string(content) != "workspace artifact" {
		t.Fatalf("unexpected file content: %q", string(content))
	}
	if result.Metadata["workspace"] != result.Workspace {
		t.Fatalf("expected workspace metadata to match result workspace")
	}
}

func TestLocalManagerTimeoutPolicy(t *testing.T) {
	manager := NewLocalManagerWithRoot(t.TempDir())
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	result, err := manager.Execute(ctx, Request{
		FlowID:       "flow-timeout",
		ActionID:     "action-timeout",
		Profile:      "terminal",
		FunctionName: "terminal",
		Input: map[string]string{
			"command": "sleep 1",
		},
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if result.ExitCode != 124 {
		t.Fatalf("expected timeout exit code 124, got %d", result.ExitCode)
	}
}

func TestLocalManagerRetriesTerminalFailures(t *testing.T) {
	manager := NewLocalManagerWithRoot(t.TempDir())
	result, err := manager.Execute(context.Background(), Request{
		FlowID:       "flow-retry",
		ActionID:     "action-retry",
		Profile:      "terminal",
		FunctionName: "terminal",
		Input: map[string]string{
			"command": "exit 3",
		},
	})
	if err == nil {
		t.Fatal("expected terminal command to fail")
	}
	if result.Attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", result.Attempts)
	}
	if result.ExitCode != 3 {
		t.Fatalf("expected exit code 3, got %d", result.ExitCode)
	}
	if result.Metadata["max_attempts"] != "2" {
		t.Fatalf("expected retry metadata to be recorded")
	}
}

func TestLocalManagerMarksPentestToolsetForSecurityGoals(t *testing.T) {
	manager := NewLocalManagerWithRoot(t.TempDir())
	result, err := manager.Execute(context.Background(), Request{
		FlowID:       "flow-pentest",
		ActionID:     "action-toolset",
		Profile:      "terminal",
		FunctionName: "terminal",
		Input: map[string]string{
			"goal": "run reconnaissance with subfinder and httpx",
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Metadata["toolset"] != toolsetPentest {
		t.Fatalf("expected pentest toolset metadata, got %s", result.Metadata["toolset"])
	}
}
