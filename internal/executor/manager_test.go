package executor

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"nyx/internal/config"
)

func TestOpenAutoFallsBackToLocalWhenDockerUnavailable(t *testing.T) {
	originalLookPath := dockerLookPath
	originalSocketPath := dockerSocketPath
	originalSocketStat := dockerSocketStat
	t.Cleanup(func() {
		dockerLookPath = originalLookPath
		dockerSocketPath = originalSocketPath
		dockerSocketStat = originalSocketStat
	})

	dockerLookPath = func(string) (string, error) {
		return "", exec.ErrNotFound
	}
	dockerSocketPath = "/tmp/nyx-test-docker.sock"

	manager, err := Open(config.Config{ExecutorMode: "auto"})
	if err != nil {
		t.Fatalf("open auto executor: %v", err)
	}
	if manager.Mode() != "local" {
		t.Fatalf("expected local fallback, got %s", manager.Mode())
	}
}

func TestOpenDockerRequiresDockerRuntime(t *testing.T) {
	originalLookPath := dockerLookPath
	originalSocketPath := dockerSocketPath
	originalSocketStat := dockerSocketStat
	t.Cleanup(func() {
		dockerLookPath = originalLookPath
		dockerSocketPath = originalSocketPath
		dockerSocketStat = originalSocketStat
	})

	dockerLookPath = func(string) (string, error) {
		return "", exec.ErrNotFound
	}
	dockerSocketPath = "/tmp/nyx-test-docker.sock"

	if _, err := Open(config.Config{ExecutorMode: "docker"}); err == nil {
		t.Fatal("expected docker mode to fail without docker runtime")
	}
}

func TestOpenDockerUsesValidatedManager(t *testing.T) {
	originalLookPath := dockerLookPath
	originalSocketPath := dockerSocketPath
	originalSocketStat := dockerSocketStat
	t.Cleanup(func() {
		dockerLookPath = originalLookPath
		dockerSocketPath = originalSocketPath
		dockerSocketStat = originalSocketStat
	})

	dockerLookPath = func(string) (string, error) { return fakeDockerBinary(t, "#!/bin/sh\nexit 0\n"), nil }
	dockerSocketPath = "/tmp/nyx-test-docker.sock"
	dockerSocketStat = func(string) (fs.FileInfo, error) {
		return fakeSocketInfo{name: "docker.sock"}, nil
	}

	manager, err := Open(config.Config{
		ExecutorMode:         "docker",
		ExecutorPentestImage: "nyx-executor-pentest:latest",
	})
	if err != nil {
		t.Fatalf("open docker executor: %v", err)
	}
	if manager.Mode() != "docker" {
		t.Fatalf("expected docker mode, got %s", manager.Mode())
	}
}

func TestDockerManagerValidateRequiresPentestImageReadiness(t *testing.T) {
	originalLookPath := dockerLookPath
	originalSocketPath := dockerSocketPath
	originalSocketStat := dockerSocketStat
	t.Cleanup(func() {
		dockerLookPath = originalLookPath
		dockerSocketPath = originalSocketPath
		dockerSocketStat = originalSocketStat
	})

	dockerLookPath = func(string) (string, error) {
		return fakeDockerBinary(t, `#!/bin/sh
if [ "$1" = "image" ] && [ "$2" = "inspect" ] && [ "$3" = "nyx-executor-pentest:latest" ]; then
	echo "Error: No such image" >&2
	exit 1
fi
exit 0
`), nil
	}
	dockerSocketPath = "/tmp/nyx-test-docker.sock"
	dockerSocketStat = func(string) (fs.FileInfo, error) {
		return fakeSocketInfo{name: "docker.sock"}, nil
	}

	manager := NewDockerManagerWithConfig("alpine:3.20", "nyx-executor-pentest:latest", t.TempDir(), "none", "", false)
	err := manager.Validate()
	if err == nil {
		t.Fatal("expected tooling image validation to fail")
	}
	if !strings.Contains(err.Error(), "make `docker-build-executor-pentest`") && !strings.Contains(err.Error(), "make docker-build-executor-pentest") {
		t.Fatalf("expected build guidance in error, got %v", err)
	}
}

func TestDockerManagerValidateRequiresCustomNetworkToExist(t *testing.T) {
	originalLookPath := dockerLookPath
	originalSocketPath := dockerSocketPath
	originalSocketStat := dockerSocketStat
	t.Cleanup(func() {
		dockerLookPath = originalLookPath
		dockerSocketPath = originalSocketPath
		dockerSocketStat = originalSocketStat
	})

	dockerLookPath = func(string) (string, error) {
		return fakeDockerBinary(t, `#!/bin/sh
if [ "$1" = "network" ] && [ "$2" = "inspect" ] && [ "$3" = "nyx-targets" ]; then
	echo "Error: No such network" >&2
	exit 1
fi
exit 0
`), nil
	}
	dockerSocketPath = "/tmp/nyx-test-docker.sock"
	dockerSocketStat = func(string) (fs.FileInfo, error) {
		return fakeSocketInfo{name: "docker.sock"}, nil
	}

	manager := NewDockerManagerWithConfig("alpine:3.20", "nyx-executor-pentest:latest", t.TempDir(), "custom", "nyx-targets", false)
	err := manager.Validate()
	if err == nil {
		t.Fatal("expected custom network validation to fail")
	}
	if !strings.Contains(err.Error(), "custom network") || !strings.Contains(err.Error(), "nyx-targets") {
		t.Fatalf("expected custom network error, got %v", err)
	}
}

func TestDockerManagerChoosesPentestImageForPentestToolset(t *testing.T) {
	manager := NewDockerManagerWithRoot("alpine:3.20", "nyx-executor-pentest:latest", t.TempDir())
	if got := manager.imageForToolset(toolsetPentest); got != "nyx-executor-pentest:latest" {
		t.Fatalf("expected pentest image, got %s", got)
	}
	if got := manager.imageForToolset(toolsetGeneral); got != "alpine:3.20" {
		t.Fatalf("expected general image, got %s", got)
	}
}

func TestDockerManagerResolveNetworkConfig(t *testing.T) {
	manager := NewDockerManagerWithConfig("alpine:3.20", "nyx-executor-pentest:latest", t.TempDir(), "bridge", "", true)

	network, err := manager.resolveNetworkConfig(map[string]string{"requires_raw_socket": "true"})
	if err != nil {
		t.Fatalf("resolve network config: %v", err)
	}
	if network.Mode != networkModeBridge {
		t.Fatalf("expected bridge mode, got %s", network.Mode)
	}
	if !network.EnableNetRaw {
		t.Fatal("expected NET_RAW to be enabled for this request")
	}
}

func TestDockerManagerRejectsRawSocketRequestWhenDisabled(t *testing.T) {
	manager := NewDockerManagerWithConfig("alpine:3.20", "nyx-executor-pentest:latest", t.TempDir(), "bridge", "", false)

	if _, err := manager.resolveNetworkConfig(map[string]string{"requires_raw_socket": "true"}); err == nil {
		t.Fatal("expected raw socket request to fail when disabled")
	}
}

func TestDockerManagerBuildDockerRunArgsUsesConfiguredNetwork(t *testing.T) {
	spec, err := resolveProfileSpec("terminal")
	if err != nil {
		t.Fatalf("resolve profile: %v", err)
	}

	args := buildDockerRunArgs("/tmp/workspace", "/workspace", spec, dockerNetworkConfig{
		Mode: networkModeCustom,
		Name: "nyx-targets",
	})
	if !reflect.DeepEqual(args[:4], []string{"run", "--rm", "--network", "nyx-targets"}) {
		t.Fatalf("unexpected docker run prefix: %v", args[:4])
	}
}

func TestDockerManagerBuildDockerRunArgsAddsNetRawCapabilityOnlyWhenRequested(t *testing.T) {
	spec, err := resolveProfileSpec("terminal")
	if err != nil {
		t.Fatalf("resolve profile: %v", err)
	}

	args := buildDockerRunArgs("/tmp/workspace", "/workspace", spec, dockerNetworkConfig{
		Mode:         networkModeBridge,
		EnableNetRaw: true,
	})
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--cap-add=NET_RAW") {
		t.Fatalf("expected NET_RAW capability in args: %v", args)
	}
}

func TestDockerManagerBuildDockerRunArgsEnforcesReadonlyResourceLimits(t *testing.T) {
	spec, err := resolveProfileSpec("terminal")
	if err != nil {
		t.Fatalf("resolve profile: %v", err)
	}

	args := buildDockerRunArgs("/tmp/workspace", "/workspace", spec, dockerNetworkConfig{
		Mode: networkModeNone,
	})
	joined := strings.Join(args, " ")
	for _, fragment := range []string{
		"--read-only",
		"--tmpfs /tmp:rw,noexec,nosuid,size=64m",
		"--cap-drop=ALL",
		"--security-opt=no-new-privileges",
		"--cpus " + spec.CPUQuota,
		"--memory " + spec.MemoryLimit,
		"--pids-limit " + strconv.Itoa(spec.PidsLimit),
	} {
		if !strings.Contains(joined, fragment) {
			t.Fatalf("expected %q in docker args: %v", fragment, args)
		}
	}
}

func TestSanitizeRelativePathRejectsTraversal(t *testing.T) {
	for _, value := range []string{"../etc/passwd", ".."} {
		if _, err := sanitizeRelativePath(value); err == nil {
			t.Fatalf("expected traversal rejection for %q", value)
		}
	}

	got, err := sanitizeRelativePath("nested/output.txt")
	if err != nil {
		t.Fatalf("sanitizeRelativePath: %v", err)
	}
	if got != "nested/output.txt" {
		t.Fatalf("unexpected sanitized path: %q", got)
	}
}

func fakeDockerBinary(t *testing.T, script string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "docker")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake docker binary: %v", err)
	}
	return path
}

type fakeSocketInfo struct {
	name string
}

func (i fakeSocketInfo) Name() string       { return i.name }
func (i fakeSocketInfo) Size() int64        { return 0 }
func (i fakeSocketInfo) Mode() fs.FileMode  { return fs.ModeSocket | 0o666 }
func (i fakeSocketInfo) ModTime() time.Time { return time.Unix(0, 0) }
func (i fakeSocketInfo) IsDir() bool        { return false }
func (i fakeSocketInfo) Sys() any           { return nil }
