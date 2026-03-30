package executor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"nyx/internal/config"
)

type Request struct {
	FlowID       string
	ActionID     string
	Profile      string
	FunctionName string
	Input        map[string]string
}

type Result struct {
	Runtime   string
	Summary   string
	Metadata  map[string]string
	Stdout    string
	Stderr    string
	Workspace string
	ExitCode  int
	Duration  time.Duration
	Attempts  int
}

type Manager interface {
	Execute(context.Context, Request) (Result, error)
	Mode() string
}

func Open(cfg config.Config) (Manager, error) {
	mode := strings.ToLower(strings.TrimSpace(cfg.ExecutorMode))
	switch mode {
	case "docker":
		manager := NewDockerManagerWithConfig(
			cfg.ExecutorImage,
			cfg.ExecutorPentestImage,
			cfg.ExecutorWorkspaceRoot,
			cfg.ExecutorNetworkMode,
			cfg.ExecutorNetworkName,
			cfg.ExecutorEnableNetRaw,
		)
		if err := manager.Validate(); err != nil {
			return nil, err
		}
		return manager, nil
	case "auto":
		manager := NewDockerManagerWithConfig(
			cfg.ExecutorImage,
			cfg.ExecutorPentestImage,
			cfg.ExecutorWorkspaceRoot,
			cfg.ExecutorNetworkMode,
			cfg.ExecutorNetworkName,
			cfg.ExecutorEnableNetRaw,
		)
		if err := manager.Validate(); err == nil {
			return manager, nil
		}
		return NewLocalManagerWithRoot(cfg.ExecutorWorkspaceRoot), nil
	default:
		if mode != "" && mode != "local" {
			return nil, fmt.Errorf("unsupported executor mode %q", cfg.ExecutorMode)
		}
		return NewLocalManagerWithRoot(cfg.ExecutorWorkspaceRoot), nil
	}
}

func defaultSummary(req Request) string {
	switch req.Profile {
	case "terminal":
		goal := strings.TrimSpace(req.Input["goal"])
		if goal == "" {
			goal = "execute controlled validation steps"
		}
		return fmt.Sprintf("Prepared an isolated terminal execution profile to %s.", goal)
	case "file":
		path := strings.TrimSpace(req.Input["path"])
		if path == "" {
			path = "workspace artifact"
		}
		return fmt.Sprintf("Prepared an isolated file execution profile for %s.", path)
	default:
		return fmt.Sprintf("Prepared an isolated %s execution profile.", req.Profile)
	}
}
