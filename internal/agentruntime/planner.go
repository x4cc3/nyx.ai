package agentruntime

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"nyx/internal/domain"
)

type PlannerMetadata struct {
	Model            string
	ResponseID       string
	ReasoningSummary string
}

type PlannerRequest struct {
	Flow               domain.Flow
	MemorySummary      string
	AvailableFunctions []domain.FunctionDef
}

type Planner interface {
	Plan(context.Context, PlannerRequest) (FlowPlan, PlannerMetadata, error)
}

type Option func(*Runtime)

func WithPlanner(planner Planner) Option {
	return func(runtime *Runtime) {
		runtime.planner = planner
	}
}

func WithPromptLibrary(library PromptLibrary) Option {
	return func(runtime *Runtime) {
		runtime.prompts = library
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(runtime *Runtime) {
		if logger != nil {
			runtime.logger = logger
		}
	}
}

func sanitizePlan(plan FlowPlan, defs []domain.FunctionDef, fallback FlowPlan) FlowPlan {
	allowedFunctions := make(map[string]struct{}, len(defs))
	for _, def := range defs {
		allowedFunctions[def.Name] = struct{}{}
	}

	sanitized := FlowPlan{Tasks: make([]TaskSpec, 0, len(plan.Tasks))}
	for _, task := range plan.Tasks {
		if strings.TrimSpace(task.Name) == "" {
			continue
		}
		safeTask := TaskSpec{
			Role:        normalizeRole(task.Role),
			Name:        strings.TrimSpace(task.Name),
			Description: strings.TrimSpace(task.Description),
			Subtasks:    make([]SubtaskSpec, 0, len(task.Subtasks)),
		}
		for _, subtask := range task.Subtasks {
			if strings.TrimSpace(subtask.Name) == "" {
				continue
			}
			safeSubtask := SubtaskSpec{
				Role:         normalizeRole(subtask.Role),
				Name:         strings.TrimSpace(subtask.Name),
				Description:  strings.TrimSpace(subtask.Description),
				MaxAttempts:  clamp(subtask.MaxAttempts, 1, 3),
				EscalateTo:   normalizeRole(subtask.EscalateTo),
				EscalateText: strings.TrimSpace(subtask.EscalateText),
				Steps:        make([]ActionSpec, 0, len(subtask.Steps)),
			}
			if safeSubtask.Role == "" {
				safeSubtask.Role = safeTask.Role
			}
			if safeSubtask.EscalateTo == "" {
				safeSubtask.EscalateTo = "planner"
			}
			for _, step := range subtask.Steps {
				if _, ok := allowedFunctions[step.FunctionName]; !ok {
					continue
				}
				safeSubtask.Steps = append(safeSubtask.Steps, ActionSpec{
					FunctionName: step.FunctionName,
					Input:        trimStringMap(step.Input, 24, 2048),
				})
			}
			if len(safeSubtask.Steps) == 0 {
				continue
			}
			safeTask.Subtasks = append(safeTask.Subtasks, safeSubtask)
			if len(safeTask.Subtasks) == 4 {
				break
			}
		}
		if len(safeTask.Subtasks) == 0 {
			continue
		}
		sanitized.Tasks = append(sanitized.Tasks, safeTask)
		if len(sanitized.Tasks) == 4 {
			break
		}
	}

	if len(sanitized.Tasks) == 0 {
		return fallback
	}
	return sanitized
}

func trimStringMap(input map[string]string, maxEntries, maxValueLen int) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	count := 0
	for key, value := range input {
		if strings.TrimSpace(key) == "" {
			continue
		}
		trimmed := strings.TrimSpace(value)
		if len(trimmed) > maxValueLen {
			trimmed = trimmed[:maxValueLen]
		}
		out[strings.TrimSpace(key)] = trimmed
		count++
		if count == maxEntries {
			break
		}
	}
	return out
}

func normalizeRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "planner", "researcher", "executor", "operator":
		return strings.ToLower(strings.TrimSpace(role))
	default:
		return ""
	}
}

func clamp(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func memorySummaryFromFlow(ctx context.Context, repoSearch func(context.Context, string, string) ([]domain.Memory, error), flowID string) string {
	memories, _ := repoSearch(ctx, flowID, "target")
	parts := make([]string, 0, len(memories))
	for _, item := range memories {
		if item.Content == "" {
			continue
		}
		parts = append(parts, item.Content)
		if len(parts) == 3 {
			break
		}
	}
	return strings.Join(parts, " | ")
}

func planArtifactName(metadata PlannerMetadata) string {
	if strings.TrimSpace(metadata.Model) == "" {
		return "runtime-plan.json"
	}
	return fmt.Sprintf("runtime-plan-%s.json", strings.ReplaceAll(metadata.Model, "/", "-"))
}
