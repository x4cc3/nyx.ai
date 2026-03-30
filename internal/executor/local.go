package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"time"
)

type LocalManager struct {
	workspaceRoot string
}

func NewLocalManager() *LocalManager {
	return NewLocalManagerWithRoot(defaultWorkspaceRoot())
}

func NewLocalManagerWithRoot(workspaceRoot string) *LocalManager {
	if workspaceRoot == "" {
		workspaceRoot = defaultWorkspaceRoot()
	}
	return &LocalManager{workspaceRoot: workspaceRoot}
}

func (m *LocalManager) Execute(ctx context.Context, req Request) (Result, error) {
	selectedToolset := resolveToolset(req.Input)
	spec, err := resolveProfileSpec(req.Profile)
	if err != nil {
		return Result{
			Runtime: "go-executor-manager(local)",
			Summary: err.Error(),
			Metadata: map[string]string{
				"mode":    "local",
				"toolset": selectedToolset,
			},
		}, err
	}

	var last Result
	var lastErr error
	for attempt := 1; attempt <= spec.MaxAttempts; attempt++ {
		result, err := m.executeAttempt(ctx, req, spec, attempt)
		last = result
		lastErr = err
		if err == nil {
			return result, nil
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			break
		}
	}
	return last, lastErr
}

func (m *LocalManager) executeAttempt(ctx context.Context, req Request, spec profileSpec, attempt int) (Result, error) {
	selectedToolset := resolveToolset(req.Input)
	workspace, err := prepareWorkspace(m.workspaceRoot, req, attempt)
	if err != nil {
		return Result{}, err
	}

	env, err := executorEnv(req, workspace, workspace)
	if err != nil {
		return Result{}, err
	}

	runCtx, cancel := context.WithTimeout(ctx, spec.Timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, "sh", "-lc", spec.ShellScript)
	cmd.Dir = workspace
	cmd.Env = env

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	started := time.Now()
	err = cmd.Run()
	duration := time.Since(started)
	exitCode := exitCodeFromError(err, runCtx.Err())

	result := Result{
		Runtime:   "go-executor-manager(local)",
		Summary:   summarizeResult(req, stdout.String(), stderr.String(), err),
		Metadata:  executionMetadata("local", spec, workspace, attempt, duration, exitCode),
		Stdout:    stdout.String(),
		Stderr:    stderr.String(),
		Workspace: workspace,
		ExitCode:  exitCode,
		Duration:  duration,
		Attempts:  attempt,
	}
	result.Metadata["toolset"] = selectedToolset
	if err != nil {
		if runCtx.Err() == context.DeadlineExceeded {
			err = fmt.Errorf("%s profile timed out after %s: %w", spec.Name, spec.Timeout, runCtx.Err())
		}
		result.Summary = summarizeResult(req, stdout.String(), stderr.String(), err)
		return result, err
	}
	return result, nil
}

func (m *LocalManager) Mode() string { return "local" }

func executionMetadata(mode string, spec profileSpec, workspace string, attempt int, duration time.Duration, exitCode int) map[string]string {
	return map[string]string{
		"mode":         mode,
		"profile":      spec.Name,
		"workspace":    workspace,
		"timeout":      spec.Timeout.String(),
		"memory_limit": spec.MemoryLimit,
		"cpu_quota":    spec.CPUQuota,
		"pids_limit":   strconv.Itoa(spec.PidsLimit),
		"attempt":      strconv.Itoa(attempt),
		"max_attempts": strconv.Itoa(spec.MaxAttempts),
		"duration_ms":  strconv.FormatInt(duration.Milliseconds(), 10),
		"exit_code":    strconv.Itoa(exitCode),
	}
}

func exitCodeFromError(err error, runErr error) int {
	if err == nil {
		return 0
	}
	if runErr == context.DeadlineExceeded {
		return 124
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}
