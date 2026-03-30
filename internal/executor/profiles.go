package executor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type profileSpec struct {
	Name        string
	Timeout     time.Duration
	MaxAttempts int
	CPUQuota    string
	MemoryLimit string
	PidsLimit   int
	ShellScript string
}

func resolveProfileSpec(profile string) (profileSpec, error) {
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case "terminal":
		return profileSpec{
			Name:        "terminal",
			Timeout:     20 * time.Second,
			MaxAttempts: 2,
			CPUQuota:    "1.0",
			MemoryLimit: "256m",
			PidsLimit:   128,
			ShellScript: `
if [ -n "${NYX_COMMAND:-}" ]; then
  sh -lc "$NYX_COMMAND"
else
  printf 'NYX terminal profile ready: %s\n' "${NYX_GOAL:-execute controlled validation steps}"
fi
`,
		}, nil
	case "file":
		return profileSpec{
			Name:        "file",
			Timeout:     10 * time.Second,
			MaxAttempts: 1,
			CPUQuota:    "0.50",
			MemoryLimit: "128m",
			PidsLimit:   64,
			ShellScript: `
target="${NYX_FILE_PATH:-artifact.txt}"
case "${NYX_FILE_OPERATION:-write}" in
  write)
    mkdir -p "$(dirname -- "$target")"
    printf '%s' "${NYX_FILE_CONTENT:-NYX file profile initialized.}" > "$target"
    printf 'wrote %s\n' "$target"
    ;;
  read)
    cat -- "$target"
    ;;
  list)
    find . -maxdepth 4 -type f | LC_ALL=C sort
    ;;
  *)
    printf 'unsupported file operation: %s\n' "$NYX_FILE_OPERATION" >&2
    exit 2
    ;;
esac
`,
		}, nil
	default:
		return profileSpec{}, fmt.Errorf("unsupported executor profile %q", profile)
	}
}

func defaultWorkspaceRoot() string {
	return filepath.Join(os.TempDir(), "nyx-executor")
}

func sanitizeID(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "unknown"
	}

	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}

	out := strings.Trim(b.String(), "._-")
	if out == "" {
		return "unknown"
	}
	return out
}

func prepareWorkspace(root string, req Request, attempt int) (string, error) {
	if strings.TrimSpace(root) == "" {
		root = defaultWorkspaceRoot()
	}

	workspace := filepath.Join(
		root,
		sanitizeID(req.FlowID),
		sanitizeID(req.ActionID),
		fmt.Sprintf("attempt-%02d", attempt),
	)
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		return "", err
	}
	return workspace, nil
}

func executorEnv(req Request, workspacePath, homePath string) ([]string, error) {
	if strings.TrimSpace(workspacePath) == "" {
		return nil, fmt.Errorf("workspace path is required")
	}
	if strings.TrimSpace(homePath) == "" {
		homePath = workspacePath
	}

	path := os.Getenv("PATH")
	if path == "" {
		path = "/usr/bin:/bin"
	}

	env := []string{
		"PATH=" + path,
		"LANG=C.UTF-8",
		"HOME=" + homePath,
		"NYX_FLOW_ID=" + req.FlowID,
		"NYX_ACTION_ID=" + req.ActionID,
		"NYX_PROFILE=" + req.Profile,
		"NYX_FUNCTION=" + req.FunctionName,
		"NYX_WORKSPACE=" + workspacePath,
	}
	if goal := strings.TrimSpace(req.Input["goal"]); goal != "" {
		env = append(env, "NYX_GOAL="+goal)
	}
	if command := strings.TrimSpace(req.Input["command"]); command != "" {
		env = append(env, "NYX_COMMAND="+command)
	}

	if strings.EqualFold(req.Profile, "file") {
		pathValue, err := sanitizeRelativePath(req.Input["path"])
		if err != nil {
			return nil, err
		}
		operation, err := normalizeFileOperation(req.Input["operation"])
		if err != nil {
			return nil, err
		}

		env = append(env,
			"NYX_FILE_PATH="+pathValue,
			"NYX_FILE_OPERATION="+operation,
			"NYX_FILE_CONTENT="+req.Input["content"],
		)
	}

	return env, nil
}

func sanitizeRelativePath(raw string) (string, error) {
	clean := filepath.Clean(strings.TrimSpace(raw))
	if clean == "." || clean == "" {
		return "artifact.txt", nil
	}
	if filepath.IsAbs(clean) {
		return "", fmt.Errorf("absolute paths are not allowed in file profile")
	}
	if clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes executor workspace")
	}
	return clean, nil
}

func normalizeFileOperation(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return "write", nil
	}
	switch value {
	case "write", "read", "list":
		return value, nil
	default:
		return "", fmt.Errorf("unsupported file operation %q", raw)
	}
}

func summarizeResult(req Request, stdout, stderr string, err error) string {
	for _, candidate := range []string{stdout, stderr} {
		lines := strings.Split(candidate, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				return line
			}
		}
	}
	if err != nil {
		return fmt.Sprintf("Execution failed for the %s profile.", req.Profile)
	}
	return defaultSummary(req)
}
