package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type DockerManager struct {
	image         string
	pentestImage  string
	workspaceRoot string
	networkMode   string
	networkName   string
	enableNetRaw  bool
	binaryPath    string
}

func NewDockerManager(image, pentestImage string) *DockerManager {
	return NewDockerManagerWithConfig(image, pentestImage, defaultWorkspaceRoot(), networkModeNone, "", false)
}

func NewDockerManagerWithRoot(image, pentestImage, workspaceRoot string) *DockerManager {
	return NewDockerManagerWithConfig(image, pentestImage, workspaceRoot, networkModeNone, "", false)
}

func NewDockerManagerWithConfig(image, pentestImage, workspaceRoot, networkMode, networkName string, enableNetRaw bool) *DockerManager {
	if strings.TrimSpace(image) == "" {
		image = "alpine:3.20"
	}
	if strings.TrimSpace(pentestImage) == "" {
		pentestImage = image
	}
	if workspaceRoot == "" {
		workspaceRoot = defaultWorkspaceRoot()
	}
	return &DockerManager{
		image:         image,
		pentestImage:  pentestImage,
		workspaceRoot: workspaceRoot,
		networkMode:   networkMode,
		networkName:   networkName,
		enableNetRaw:  enableNetRaw,
	}
}

func (m *DockerManager) Validate() error {
	if _, err := m.resolveNetworkConfig(nil); err != nil {
		return err
	}
	binaryPath, err := detectDockerRuntime()
	if err != nil {
		return err
	}
	if err := m.validateToolingImage(binaryPath); err != nil {
		return err
	}
	if err := m.validateCustomNetwork(binaryPath); err != nil {
		return err
	}
	m.binaryPath = binaryPath
	return nil
}

func (m *DockerManager) Execute(ctx context.Context, req Request) (Result, error) {
	selectedToolset := resolveToolset(req.Input)
	selectedImage := m.imageForToolset(selectedToolset)
	spec, err := resolveProfileSpec(req.Profile)
	if err != nil {
		return Result{
			Runtime: "docker:" + selectedImage,
			Summary: err.Error(),
			Metadata: map[string]string{
				"mode":    "docker",
				"image":   selectedImage,
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

func (m *DockerManager) Mode() string { return "docker" }

func (m *DockerManager) executeAttempt(ctx context.Context, req Request, spec profileSpec, attempt int) (Result, error) {
	selectedToolset := resolveToolset(req.Input)
	selectedImage := m.imageForToolset(selectedToolset)
	network, err := m.resolveNetworkConfig(req.Input)
	if err != nil {
		return Result{
			Runtime: "docker:" + selectedImage,
			Summary: err.Error(),
			Metadata: map[string]string{
				"mode":         "docker",
				"image":        selectedImage,
				"toolset":      selectedToolset,
				"network_mode": normalizeNetworkMode(m.networkMode),
			},
			Attempts: attempt,
			ExitCode: -1,
		}, err
	}
	workspace, err := prepareWorkspace(m.workspaceRoot, req, attempt)
	if err != nil {
		return Result{}, err
	}

	containerWorkspace := "/workspace"
	// Keep tool home/state under the writable tmpfs even when the root filesystem is read-only.
	env, err := executorEnv(req, containerWorkspace, "/tmp")
	if err != nil {
		return Result{}, err
	}

	runCtx, cancel := context.WithTimeout(ctx, spec.Timeout)
	defer cancel()

	args := buildDockerRunArgs(workspace, containerWorkspace, spec, network)
	for _, item := range env {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 {
			continue
		}
		if parts[0] == "PATH" || parts[0] == "LANG" || parts[0] == "HOME" {
			continue
		}
		args = append(args, "-e", item)
	}
	args = append(args, selectedImage, "sh", "-lc", spec.ShellScript)

	binaryPath := strings.TrimSpace(m.binaryPath)
	if binaryPath == "" {
		if err := m.Validate(); err != nil {
			result := Result{
				Runtime: "docker:" + selectedImage,
				Summary: err.Error(),
				Metadata: map[string]string{
					"mode":    "docker",
					"image":   selectedImage,
					"toolset": selectedToolset,
				},
				Workspace: workspace,
				Attempts:  attempt,
				ExitCode:  -1,
			}
			return result, err
		}
		binaryPath = m.binaryPath
	}

	cmd := exec.CommandContext(runCtx, binaryPath, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	started := time.Now()
	err = cmd.Run()
	duration := time.Since(started)
	exitCode := exitCodeFromError(err, runCtx.Err())

	result := Result{
		Runtime:   "docker:" + selectedImage,
		Summary:   summarizeResult(req, stdout.String(), stderr.String(), err),
		Metadata:  executionMetadata("docker", spec, workspace, attempt, duration, exitCode),
		Stdout:    stdout.String(),
		Stderr:    stderr.String(),
		Workspace: workspace,
		ExitCode:  exitCode,
		Duration:  duration,
		Attempts:  attempt,
	}
	result.Metadata["image"] = selectedImage
	result.Metadata["toolset"] = selectedToolset
	result.Metadata["network_mode"] = network.Mode
	if network.Name != "" {
		result.Metadata["network_name"] = network.Name
	}
	result.Metadata["net_raw"] = strconv.FormatBool(network.EnableNetRaw)
	if err != nil {
		if runCtx.Err() == context.DeadlineExceeded {
			err = fmt.Errorf("%s profile timed out after %s: %w", spec.Name, spec.Timeout, runCtx.Err())
		}
		result.Summary = summarizeResult(req, stdout.String(), stderr.String(), err)
		return result, err
	}
	return result, nil
}

func (m *DockerManager) imageForToolset(toolset string) string {
	if toolset == toolsetPentest && strings.TrimSpace(m.pentestImage) != "" {
		return m.pentestImage
	}
	return m.image
}

func buildDockerRunArgs(workspace, containerWorkspace string, spec profileSpec, network dockerNetworkConfig) []string {
	networkTarget := network.Mode
	if network.Mode == networkModeCustom {
		networkTarget = network.Name
	}

	args := []string{
		"run",
		"--rm",
		"--network", networkTarget,
		"--cpus", spec.CPUQuota,
		"--memory", spec.MemoryLimit,
		"--pids-limit", strconv.Itoa(spec.PidsLimit),
		"--cap-drop=ALL",
		"--security-opt=no-new-privileges",
		"--read-only",
		"--tmpfs", "/tmp:rw,noexec,nosuid,size=64m",
		"--mount", "type=bind,src=" + workspace + ",dst=" + containerWorkspace,
		"-w", containerWorkspace,
		"-e", "HOME=" + containerWorkspace,
	}
	if network.EnableNetRaw {
		args = append(args, "--cap-add=NET_RAW")
	}
	return args
}

var (
	dockerLookPath   = exec.LookPath
	dockerSocketPath = "/var/run/docker.sock"
	dockerSocketStat = os.Stat
)

func detectDockerRuntime() (string, error) {
	binaryPath, err := dockerLookPath("docker")
	if err != nil {
		return "", errors.New("docker executor mode requires the docker CLI in PATH")
	}
	info, err := dockerSocketStat(dockerSocketPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("docker executor mode requires %s to be mounted", dockerSocketPath)
		}
		return "", fmt.Errorf("inspect docker socket %s: %w", dockerSocketPath, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("docker executor mode expected %s to be a socket, not a directory", dockerSocketPath)
	}
	if info.Mode()&os.ModeSocket == 0 {
		return "", fmt.Errorf("docker executor mode expected %s to be a unix socket", dockerSocketPath)
	}
	return binaryPath, nil
}

func (m *DockerManager) validateToolingImage(binaryPath string) error {
	image := strings.TrimSpace(m.imageForToolset(toolsetPentest))
	if image == "" {
		return nil
	}
	stderr, err := dockerProbe(binaryPath, "image", "inspect", image)
	if err == nil {
		return nil
	}
	message := fmt.Sprintf("docker executor mode requires tooling image %q to be ready", image)
	if image == strings.TrimSpace(m.pentestImage) {
		message += "; build it with `make docker-build-executor-pentest` or point NYX_EXECUTOR_IMAGE_FOR_PENTEST at a ready image"
	}
	if stderr != "" {
		message += ": " + stderr
	}
	return errors.New(message)
}

func (m *DockerManager) validateCustomNetwork(binaryPath string) error {
	if normalizeNetworkMode(m.networkMode) != networkModeCustom {
		return nil
	}
	name := strings.TrimSpace(m.networkName)
	if name == "" {
		return errors.New("docker executor custom network mode requires NYX_EXECUTOR_NETWORK_NAME")
	}
	stderr, err := dockerProbe(binaryPath, "network", "inspect", name)
	if err == nil {
		return nil
	}
	message := fmt.Sprintf("docker executor mode requires custom network %q to exist before startup", name)
	if stderr != "" {
		message += ": " + stderr
	}
	return errors.New(message)
}

func dockerProbe(binaryPath string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("probe timed out for docker %s", strings.Join(args, " "))
		}
		return strings.TrimSpace(stderr.String()), err
	}
	return "", nil
}
