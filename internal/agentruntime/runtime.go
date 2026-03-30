package agentruntime

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"nyx/internal/domain"
	"nyx/internal/functions"
	"nyx/internal/store"
)

var errOperatorInputRequired = errors.New("operator input required")

type ActionInvocation struct {
	FlowID       string
	TaskID       string
	SubtaskID    string
	Role         string
	FunctionName string
	Input        map[string]string
}

type ActionExecutor func(context.Context, ActionInvocation) functions.CallResult
type EventPublisher func(ctx context.Context, flowID, eventType, message string, payload map[string]any)

type Runtime struct {
	repo         store.Repository
	defs         []domain.FunctionDef
	execute      ActionExecutor
	publish      EventPublisher
	prompts      PromptLibrary
	planner      Planner
	actionPolicy ActionPolicy
	logger       *slog.Logger
}

type FlowPlan struct {
	Tasks []TaskSpec
}

type TaskSpec struct {
	Role        string
	Name        string
	Description string
	Subtasks    []SubtaskSpec
}

type SubtaskSpec struct {
	Role         string
	Name         string
	Description  string
	MaxAttempts  int
	EscalateTo   string
	EscalateText string
	Steps        []ActionSpec
}

type ActionSpec struct {
	FunctionName string
	Input        map[string]string
}

func New(repo store.Repository, defs []domain.FunctionDef, execute ActionExecutor, publish EventPublisher, opts ...Option) *Runtime {
	if publish == nil {
		publish = func(context.Context, string, string, string, map[string]any) {}
	}
	runtime := &Runtime{
		repo:    repo,
		defs:    defs,
		execute: execute,
		publish: publish,
		prompts: DefaultPromptLibrary("gpt-5.1-codex-mini"),
		logger:  slog.Default(),
	}
	for _, opt := range opts {
		opt(runtime)
	}
	return runtime
}

func (r *Runtime) RunFlow(ctx context.Context, flow domain.Flow) error {
	r.publish(ctx, flow.ID, domain.EventFlowStatus, "Flow execution started", map[string]any{
		"status": flow.Status,
		"mode":   "agent-runtime",
	})

	plan, err := r.planFlow(ctx, flow)
	if err != nil {
		if _, err := r.repo.UpdateFlowStatus(ctx, flow.ID, domain.StatusFailed); err != nil {
			r.logger.Warn("failed to update flow status", "flow_id", flow.ID, "err", err)
		}
		r.publish(ctx, flow.ID, domain.EventFlowStatus, "Flow execution failed", map[string]any{
			"status": domain.StatusFailed,
			"error":  err.Error(),
		})
		return err
	}

	r.publish(ctx, flow.ID, domain.EventRuntimePlanReady, "Agent runtime decomposed the flow into specialist tasks", map[string]any{
		"tasks": len(plan.Tasks),
	})

	for _, task := range plan.Tasks {
		if err := r.runTask(ctx, flow, task); err != nil {
			if _, err := r.repo.UpdateFlowStatus(ctx, flow.ID, domain.StatusFailed); err != nil {
				r.logger.Warn("failed to update flow status", "flow_id", flow.ID, "err", err)
			}
			r.publish(ctx, flow.ID, domain.EventFlowStatus, "Flow execution failed", map[string]any{
				"status": domain.StatusFailed,
				"error":  err.Error(),
				"task":   task.Name,
			})
			return err
		}
	}

	if _, err := r.repo.AddFinding(ctx, flow.ID, "Agent runtime completed", "info", "NYX v2 completed a supervised agent-runtime flow with decomposition, retries, and escalation rules."); err != nil {
		r.logger.Warn("failed to add finding", "flow_id", flow.ID, "err", err)
	}
	flow, err = r.repo.UpdateFlowStatus(ctx, flow.ID, domain.StatusCompleted)
	if err != nil {
		r.logger.Warn("failed to update flow status", "flow_id", flow.ID, "err", err)
	}
	r.publish(ctx, flow.ID, domain.EventFlowStatus, "Flow execution completed", map[string]any{
		"status": flow.Status,
		"mode":   "agent-runtime",
	})
	return nil
}

func (r *Runtime) planFlow(ctx context.Context, flow domain.Flow) (FlowPlan, error) {
	plannerTemplate := r.prompts.Template("planner")
	agent, err := r.repo.CreateAgent(ctx, flow.ID, plannerTemplate.Role, plannerTemplate.Model)
	if err != nil {
		return FlowPlan{}, err
	}
	if err := r.repo.UpdateAgentStatus(ctx, agent.ID, domain.StatusRunning); err != nil {
		r.logger.Warn("failed to update agent status", "agent_id", agent.ID, "err", err)
	}
	r.publish(ctx, flow.ID, domain.EventAgentStarted, "Planner agent activated", map[string]any{
		"agent_id": agent.ID,
		"role":     agent.Role,
	})

	task, err := r.repo.CreateTask(ctx, flow.ID, "Mission planning", "Decompose the engagement into specialist tasks with runtime supervision rules.", agent.Role)
	if err != nil {
		if err := r.repo.CompleteAgent(ctx, agent.ID, domain.StatusFailed); err != nil {
			r.logger.Warn("failed to complete agent", "agent_id", agent.ID, "err", err)
		}
		return FlowPlan{}, err
	}
	if err := r.repo.UpdateTaskStatus(ctx, task.ID, domain.StatusRunning); err != nil {
		r.logger.Warn("failed to update task status", "task_id", task.ID, "err", err)
	}

	subtask, err := r.repo.CreateSubtask(ctx, flow.ID, task.ID, "Initial decomposition", "Gather baseline context and prepare the execution tree.", agent.Role)
	if err != nil {
		if err := r.repo.CompleteTask(ctx, task.ID, domain.StatusFailed); err != nil {
			r.logger.Warn("failed to complete task", "task_id", task.ID, "err", err)
		}
		if err := r.repo.CompleteAgent(ctx, agent.ID, domain.StatusFailed); err != nil {
			r.logger.Warn("failed to complete agent", "agent_id", agent.ID, "err", err)
		}
		return FlowPlan{}, err
	}
	if err := r.repo.UpdateSubtaskStatus(ctx, subtask.ID, domain.StatusRunning); err != nil {
		r.logger.Warn("failed to update subtask status", "subtask_id", subtask.ID, "err", err)
	}

	planningAction, err := r.repo.CreateAction(ctx, flow.ID, task.ID, subtask.ID, agent.Role, "done", "planner-control", map[string]string{
		"stage": "planning",
	})
	if err != nil {
		r.logger.Warn("failed to create action", "flow_id", flow.ID, "err", err)
	}
	planningExecution, err := r.repo.CreateExecution(ctx, flow.ID, planningAction.ID, "planner", plannerTemplate.Model, map[string]string{
		"function_name": "done",
		"agent_role":    agent.Role,
		"stage":         "planning",
	})

	fallbackPlan := r.enrichPlan(flow, r.decompose(flow))
	memorySummary := memorySummaryFromFlow(ctx, r.repo.SearchMemories, flow.ID)
	plan := fallbackPlan
	metadata := PlannerMetadata{Model: plannerTemplate.Model}
	if r.planner != nil {
		nextPlan, nextMetadata, err := r.planner.Plan(ctx, PlannerRequest{
			Flow:               flow,
			MemorySummary:      memorySummary,
			AvailableFunctions: r.defs,
		})
		if err != nil {
			if _, completeErr := r.repo.CompleteAction(ctx, planningAction.ID, domain.StatusFailed, map[string]string{
				"summary": "Planner model request failed.",
				"error":   err.Error(),
			}); completeErr != nil {
				r.logger.Warn("failed to complete action", "action_id", planningAction.ID, "err", completeErr)
			}
			if completeErr := r.repo.CompleteExecution(ctx, planningExecution.ID, domain.StatusFailed, "planner", metadata.Model, map[string]string{
				"function_name": "done",
				"agent_role":    agent.Role,
				"stage":         "planning",
				"error":         err.Error(),
			}); completeErr != nil {
				r.logger.Warn("failed to complete execution", "execution_id", planningExecution.ID, "err", completeErr)
			}
			if completeErr := r.repo.CompleteSubtask(ctx, subtask.ID, domain.StatusFailed); completeErr != nil {
				r.logger.Warn("failed to complete subtask", "subtask_id", subtask.ID, "err", completeErr)
			}
			if completeErr := r.repo.CompleteTask(ctx, task.ID, domain.StatusFailed); completeErr != nil {
				r.logger.Warn("failed to complete task", "task_id", task.ID, "err", completeErr)
			}
			if completeErr := r.repo.CompleteAgent(ctx, agent.ID, domain.StatusFailed); completeErr != nil {
				r.logger.Warn("failed to complete agent", "agent_id", agent.ID, "err", completeErr)
			}
			return FlowPlan{}, fmt.Errorf("planner openai request failed: %w", err)
		}
		metadata = nextMetadata
		if strings.TrimSpace(metadata.Model) == "" {
			metadata.Model = plannerTemplate.Model
		}
		plan = r.enrichPlan(flow, sanitizePlan(nextPlan, r.defs, fallbackPlan))
		if _, err := r.repo.AddArtifact(ctx, flow.ID, planningAction.ID, domain.ArtifactKindPlan, planArtifactName(metadata), renderPlan(plan), map[string]string{
			"model":       metadata.Model,
			"response_id": metadata.ResponseID,
		}); err != nil {
			r.logger.Warn("failed to add artifact", "flow_id", flow.ID, "action_id", planningAction.ID, "err", err)
		}
		if strings.TrimSpace(metadata.ReasoningSummary) != "" {
			if _, err := r.repo.AddMemory(ctx, flow.ID, planningAction.ID, domain.MemoryKindPlanSummary, metadata.ReasoningSummary, map[string]string{
				"agent_role": "planner",
				"model":      metadata.Model,
				"namespace":  domain.MemoryNamespaceOperatorNotes,
			}); err != nil {
				r.logger.Warn("failed to add memory", "flow_id", flow.ID, "action_id", planningAction.ID, "err", err)
			}
		}
		r.publish(ctx, flow.ID, domain.EventRuntimePlanGenerated, "Planner model generated a structured flow plan", map[string]any{
			"model":       metadata.Model,
			"response_id": metadata.ResponseID,
			"tasks":       len(plan.Tasks),
		})
	}
	if _, err := r.repo.CompleteAction(ctx, planningAction.ID, domain.StatusCompleted, map[string]string{
		"summary": "Planner completed structured runtime decomposition.",
		"model":   metadata.Model,
	}); err != nil {
		r.logger.Warn("failed to complete action", "action_id", planningAction.ID, "err", err)
	}
	if err := r.repo.CompleteExecution(ctx, planningExecution.ID, domain.StatusCompleted, "planner", metadata.Model, map[string]string{
		"function_name": "done",
		"agent_role":    agent.Role,
		"stage":         "planning",
		"model":         metadata.Model,
		"tasks":         fmt.Sprintf("%d", len(plan.Tasks)),
	}); err != nil {
		r.logger.Warn("failed to complete execution", "execution_id", planningExecution.ID, "err", err)
	}

	if err := r.repo.CompleteSubtask(ctx, subtask.ID, domain.StatusCompleted); err != nil {
		r.logger.Warn("failed to complete subtask", "subtask_id", subtask.ID, "err", err)
	}
	if err := r.repo.CompleteTask(ctx, task.ID, domain.StatusCompleted); err != nil {
		r.logger.Warn("failed to complete task", "task_id", task.ID, "err", err)
	}
	if err := r.repo.CompleteAgent(ctx, agent.ID, domain.StatusCompleted); err != nil {
		r.logger.Warn("failed to complete agent", "agent_id", agent.ID, "err", err)
	}
	r.publish(ctx, flow.ID, domain.EventAgentCompleted, "Planner agent completed decomposition", map[string]any{
		"agent_id": agent.ID,
		"role":     agent.Role,
		"tasks":    len(plan.Tasks),
		"model":    metadata.Model,
	})
	return plan, nil
}

func (r *Runtime) runTask(ctx context.Context, flow domain.Flow, spec TaskSpec) error {
	template := r.prompts.Template(spec.Role)
	agent, err := r.repo.CreateAgent(ctx, flow.ID, spec.Role, template.Model)
	if err != nil {
		return err
	}
	if err := r.repo.UpdateAgentStatus(ctx, agent.ID, domain.StatusRunning); err != nil {
		r.logger.Warn("failed to update agent status", "agent_id", agent.ID, "err", err)
	}
	r.publish(ctx, flow.ID, domain.EventAgentStarted, "Specialist agent activated", map[string]any{
		"agent_id": agent.ID,
		"role":     agent.Role,
		"task":     spec.Name,
	})

	task, err := r.repo.CreateTask(ctx, flow.ID, spec.Name, spec.Description, spec.Role)
	if err != nil {
		if err := r.repo.CompleteAgent(ctx, agent.ID, domain.StatusFailed); err != nil {
			r.logger.Warn("failed to complete agent", "agent_id", agent.ID, "err", err)
		}
		return err
	}
	if err := r.repo.UpdateTaskStatus(ctx, task.ID, domain.StatusRunning); err != nil {
		r.logger.Warn("failed to update task status", "task_id", task.ID, "err", err)
	}
	r.publish(ctx, flow.ID, domain.EventTaskCreated, "Runtime task created", map[string]any{
		"task_id":     task.ID,
		"agent_role":  spec.Role,
		"subtasks":    len(spec.Subtasks),
		"task_name":   spec.Name,
		"agent_model": template.Model,
	})

	for _, subtaskSpec := range spec.Subtasks {
		if err := r.runSubtask(ctx, flow, task.ID, spec, subtaskSpec); err != nil {
			if err := r.repo.CompleteTask(ctx, task.ID, domain.StatusFailed); err != nil {
				r.logger.Warn("failed to complete task", "task_id", task.ID, "err", err)
			}
			if err := r.repo.CompleteAgent(ctx, agent.ID, domain.StatusFailed); err != nil {
				r.logger.Warn("failed to complete agent", "agent_id", agent.ID, "err", err)
			}
			return err
		}
	}

	if err := r.repo.CompleteTask(ctx, task.ID, domain.StatusCompleted); err != nil {
		r.logger.Warn("failed to complete task", "task_id", task.ID, "err", err)
	}
	if err := r.repo.CompleteAgent(ctx, agent.ID, domain.StatusCompleted); err != nil {
		r.logger.Warn("failed to complete agent", "agent_id", agent.ID, "err", err)
	}
	r.publish(ctx, flow.ID, domain.EventAgentCompleted, "Specialist agent completed assigned task", map[string]any{
		"agent_id": agent.ID,
		"role":     agent.Role,
		"task_id":  task.ID,
	})
	return nil
}

func (r *Runtime) runSubtask(ctx context.Context, flow domain.Flow, taskID string, task TaskSpec, spec SubtaskSpec) error {
	role := spec.Role
	if role == "" {
		role = task.Role
	}
	maxAttempts := spec.MaxAttempts
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	subtask, err := r.repo.CreateSubtask(ctx, flow.ID, taskID, spec.Name, spec.Description, role)
	if err != nil {
		return err
	}
	if err := r.repo.UpdateSubtaskStatus(ctx, subtask.ID, domain.StatusRunning); err != nil {
		r.logger.Warn("failed to update subtask status", "subtask_id", subtask.ID, "err", err)
	}
	r.publish(ctx, flow.ID, domain.EventSubtaskStarted, "Runtime subtask started", map[string]any{
		"task_id":      taskID,
		"subtask_id":   subtask.ID,
		"agent_role":   role,
		"max_attempts": maxAttempts,
	})

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		r.publish(ctx, flow.ID, domain.EventSubtaskAttempt, "Runtime subtask attempt started", map[string]any{
			"task_id":    taskID,
			"subtask_id": subtask.ID,
			"attempt":    attempt,
		})

		var failedResult functions.CallResult
		var failedStep ActionSpec
		success := true
		if r.actionPolicy != nil {
			var runErr error
			failedStep, failedResult, runErr = r.runPolicyAttempt(ctx, flow, taskID, subtask.ID, task, spec, role, attempt)
			success = runErr == nil
		} else {
			for _, step := range spec.Steps {
				result := r.executeAction(ctx, flow, taskID, subtask.ID, role, task.Name, spec.Name, step)
				if result.Err != nil {
					failedResult = result
					failedStep = step
					success = false
					break
				}
			}
		}
		if success {
			if err := r.repo.CompleteSubtask(ctx, subtask.ID, domain.StatusCompleted); err != nil {
				r.logger.Warn("failed to complete subtask", "subtask_id", subtask.ID, "err", err)
			}
			r.publish(ctx, flow.ID, domain.EventSubtaskCompleted, "Runtime subtask completed", map[string]any{
				"task_id":    taskID,
				"subtask_id": subtask.ID,
				"attempts":   attempt,
			})
			return nil
		}

		if errors.Is(failedResult.Err, errOperatorInputRequired) || functions.ApprovalRequired(failedResult) {
			if err := r.repo.CompleteSubtask(ctx, subtask.ID, domain.StatusFailed); err != nil {
				r.logger.Warn("failed to complete subtask", "subtask_id", subtask.ID, "err", err)
			}
			r.publish(ctx, flow.ID, domain.EventSubtaskOperatorInputRequired, "Runtime subtask paused for operator approval or input", map[string]any{
				"task_id":       taskID,
				"subtask_id":    subtask.ID,
				"attempt":       attempt,
				"function_name": failedStep.FunctionName,
				"error":         failureText(failedResult),
			})
			return fmt.Errorf("%s requested operator input", spec.Name)
		}

		if attempt < maxAttempts {
			errText := "subtask execution failed"
			if failedResult.Err != nil {
				errText = failedResult.Err.Error()
			}
			r.publish(ctx, flow.ID, domain.EventSubtaskRetry, "Runtime subtask will retry after a failed action", map[string]any{
				"task_id":       taskID,
				"subtask_id":    subtask.ID,
				"attempt":       attempt,
				"next_attempt":  attempt + 1,
				"function_name": failedStep.FunctionName,
				"error":         errText,
			})
			continue
		}

		if err := r.repo.CompleteSubtask(ctx, subtask.ID, domain.StatusFailed); err != nil {
			r.logger.Warn("failed to complete subtask", "subtask_id", subtask.ID, "err", err)
		}
		if _, err := r.repo.AddFinding(ctx, flow.ID, "Subtask escalation required", "medium", fmt.Sprintf("%s failed after %d attempts during %s.", spec.Name, maxAttempts, task.Name)); err != nil {
			r.logger.Warn("failed to add finding", "flow_id", flow.ID, "err", err)
		}
		r.publish(ctx, flow.ID, domain.EventSubtaskFailed, "Runtime subtask exhausted retries", map[string]any{
			"task_id":       taskID,
			"subtask_id":    subtask.ID,
			"attempts":      maxAttempts,
			"function_name": failedStep.FunctionName,
			"error":         failureText(failedResult),
		})
		if err := r.escalateFailure(ctx, flow, task, spec, failedResult.Err); err != nil {
			return fmt.Errorf("%s failed after escalation: %w", spec.Name, err)
		}
		// Escalation succeeded — operator was notified. Continue the flow
		// rather than aborting so that subsequent tasks can still execute.
		return nil
	}

	return nil
}

func (r *Runtime) runPolicyAttempt(ctx context.Context, flow domain.Flow, taskID, subtaskID string, task TaskSpec, spec SubtaskSpec, role string, attempt int) (ActionSpec, functions.CallResult, error) {
	observations := make([]ActionObservation, 0, max(1, len(spec.Steps)+2))
	turnBudget := policyTurnBudget(spec)
	var lastStep ActionSpec
	var lastResult functions.CallResult
	for turn := 1; turn <= turnBudget; turn++ {
		step, meta, err := r.nextPolicyAction(ctx, flow, task, spec, attempt, turn, observations)
		if err != nil {
			return fallbackActionSpec(spec, turn), functions.CallResult{
				Profile: "control",
				Runtime: "go-openai-policy",
				Output: map[string]string{
					"summary": "Action policy request failed.",
					"error":   err.Error(),
				},
				Err: err,
			}, err
		}
		if step.Input == nil {
			step.Input = map[string]string{}
		}
		if meta.Model != "" {
			step.Input["_nyx_policy_model"] = meta.Model
		}
		if meta.ResponseID != "" {
			step.Input["_nyx_policy_response_id"] = meta.ResponseID
		}
		if meta.ReasoningSummary != "" {
			step.Input["_nyx_policy_reasoning"] = meta.ReasoningSummary
		}
		r.publish(ctx, flow.ID, domain.EventActionDecided, "Action policy selected the next function call", map[string]any{
			"task_id":       taskID,
			"subtask_id":    subtaskID,
			"attempt":       attempt,
			"turn":          turn,
			"function_name": step.FunctionName,
			"model":         meta.Model,
		})
		if repeatedToolCallExceeded(observations, step) {
			result := functions.CallResult{
				Profile: "control",
				Runtime: "go-agent-runtime",
				Output: map[string]string{
					"summary": "Action policy repeated the same tool call without new direction.",
					"error":   "repeated tool call budget exhausted",
				},
				Err: fmt.Errorf("action policy repeated %s without new direction", step.FunctionName),
			}
			r.publish(ctx, flow.ID, domain.EventActionRepetitionBlocked, "Action policy repeated the same tool call too many times", map[string]any{
				"task_id":       taskID,
				"subtask_id":    subtaskID,
				"attempt":       attempt,
				"turn":          turn,
				"function_name": step.FunctionName,
			})
			return step, result, result.Err
		}
		result := r.executeAction(ctx, flow, taskID, subtaskID, role, task.Name, spec.Name, step)
		lastStep = step
		lastResult = result
		observations = append(observations, ActionObservation{
			FunctionName: step.FunctionName,
			Input:        cloneStringMap(step.Input),
			Output:       cloneStringMap(result.Output),
			Error:        errorString(result.Err),
		})
		if result.Err != nil {
			return step, result, result.Err
		}
		if step.FunctionName == "done" {
			return step, result, nil
		}
		if step.FunctionName == "ask" {
			result.Err = errOperatorInputRequired
			return step, result, result.Err
		}
	}

	lastResult.Err = fmt.Errorf("action policy exhausted %d turns without emitting done", turnBudget)
	if lastResult.Output == nil {
		lastResult.Output = map[string]string{}
	}
	lastResult.Output["summary"] = "Action policy exhausted its turn budget without completing the subtask."
	return lastStep, lastResult, lastResult.Err
}

func (r *Runtime) nextPolicyAction(ctx context.Context, flow domain.Flow, task TaskSpec, spec SubtaskSpec, attempt, turn int, observations []ActionObservation) (ActionSpec, ActionPolicyMetadata, error) {
	fallback := fallbackActionSpec(spec, turn)
	if r.actionPolicy == nil {
		return fallback, ActionPolicyMetadata{}, nil
	}
	memorySummary := memorySummaryFromFlow(ctx, r.repo.SearchMemories, flow.ID)
	decision, metadata, err := r.actionPolicy.NextAction(ctx, ActionPolicyRequest{
		Flow:               flow,
		Task:               task,
		Subtask:            spec,
		Attempt:            attempt,
		Turn:               turn,
		MemorySummary:      memorySummary,
		AvailableFunctions: r.defs,
		PlannedSteps:       spec.Steps,
		Observations:       observations,
	})
	if err != nil {
		return fallback, metadata, err
	}
	return sanitizeActionDecision(decision, r.defs, fallback), metadata, nil
}

func fallbackActionSpec(spec SubtaskSpec, turn int) ActionSpec {
	if len(spec.Steps) == 0 {
		return ActionSpec{FunctionName: "search_memory", Input: map[string]string{"query": "subtask context", "namespace": domain.MemoryNamespaceTargetObservations}}
	}
	index := turn - 1
	if index < 0 {
		index = 0
	}
	if index >= len(spec.Steps) {
		index = len(spec.Steps) - 1
	}
	return ActionSpec{
		FunctionName: spec.Steps[index].FunctionName,
		Input:        cloneStringMap(spec.Steps[index].Input),
	}
}

func policyTurnBudget(spec SubtaskSpec) int {
	budget := len(spec.Steps) + 2
	if budget < 2 {
		budget = 2
	}
	if budget > 6 {
		budget = 6
	}
	return budget
}

func failureText(result functions.CallResult) string {
	if result.Err != nil {
		return result.Err.Error()
	}
	if result.Output["error"] != "" {
		return result.Output["error"]
	}
	return "subtask execution failed"
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (r *Runtime) escalateFailure(ctx context.Context, flow domain.Flow, failedTask TaskSpec, failedSubtask SubtaskSpec, cause error) error {
	role := failedSubtask.EscalateTo
	if role == "" {
		role = "planner"
	}
	template := r.prompts.Template(role)
	agent, err := r.repo.CreateAgent(ctx, flow.ID, role, template.Model)
	if err != nil {
		return err
	}
	if err := r.repo.UpdateAgentStatus(ctx, agent.ID, domain.StatusRunning); err != nil {
		r.logger.Warn("failed to update agent status", "agent_id", agent.ID, "err", err)
	}

	task, err := r.repo.CreateTask(ctx, flow.ID, "Escalation review", "Reassign a blocked subtask to a supervisor role and request operator guidance.", role)
	if err != nil {
		if err := r.repo.CompleteAgent(ctx, agent.ID, domain.StatusFailed); err != nil {
			r.logger.Warn("failed to complete agent", "agent_id", agent.ID, "err", err)
		}
		return err
	}
	if err := r.repo.UpdateTaskStatus(ctx, task.ID, domain.StatusRunning); err != nil {
		r.logger.Warn("failed to update task status", "task_id", task.ID, "err", err)
	}

	subtask, err := r.repo.CreateSubtask(ctx, flow.ID, task.ID, "Operator escalation", "Escalate a repeated failure and request the next safe operator decision.", role)
	if err != nil {
		if err := r.repo.CompleteTask(ctx, task.ID, domain.StatusFailed); err != nil {
			r.logger.Warn("failed to complete task", "task_id", task.ID, "err", err)
		}
		if err := r.repo.CompleteAgent(ctx, agent.ID, domain.StatusFailed); err != nil {
			r.logger.Warn("failed to complete agent", "agent_id", agent.ID, "err", err)
		}
		return err
	}
	if err := r.repo.UpdateSubtaskStatus(ctx, subtask.ID, domain.StatusRunning); err != nil {
		r.logger.Warn("failed to update subtask status", "subtask_id", subtask.ID, "err", err)
	}

	r.publish(ctx, flow.ID, domain.EventTaskReassigned, "Runtime reassigned a blocked subtask for escalation", map[string]any{
		"agent_role":       role,
		"failed_task":      failedTask.Name,
		"failed_subtask":   failedSubtask.Name,
		"escalation_task":  task.ID,
		"escalation_agent": agent.ID,
	})

	message := failedSubtask.EscalateText
	if strings.TrimSpace(message) == "" {
		message = fmt.Sprintf("Repeated failure in %s/%s requires operator review.", failedTask.Name, failedSubtask.Name)
	}
	result := r.executeAction(ctx, flow, task.ID, subtask.ID, role, task.Name, subtask.Name, ActionSpec{
		FunctionName: "ask",
		Input: map[string]string{
			"reason":       cause.Error(),
			"message":      message,
			"failed_task":  failedTask.Name,
			"failed_stage": failedSubtask.Name,
		},
	})
	if result.Err != nil {
		if err := r.repo.CompleteSubtask(ctx, subtask.ID, domain.StatusFailed); err != nil {
			r.logger.Warn("failed to complete subtask", "subtask_id", subtask.ID, "err", err)
		}
		if err := r.repo.CompleteTask(ctx, task.ID, domain.StatusFailed); err != nil {
			r.logger.Warn("failed to complete task", "task_id", task.ID, "err", err)
		}
		if err := r.repo.CompleteAgent(ctx, agent.ID, domain.StatusFailed); err != nil {
			r.logger.Warn("failed to complete agent", "agent_id", agent.ID, "err", err)
		}
		return result.Err
	}

	if err := r.repo.CompleteSubtask(ctx, subtask.ID, domain.StatusCompleted); err != nil {
		r.logger.Warn("failed to complete subtask", "subtask_id", subtask.ID, "err", err)
	}
	if err := r.repo.CompleteTask(ctx, task.ID, domain.StatusCompleted); err != nil {
		r.logger.Warn("failed to complete task", "task_id", task.ID, "err", err)
	}
	if err := r.repo.CompleteAgent(ctx, agent.ID, domain.StatusCompleted); err != nil {
		r.logger.Warn("failed to complete agent", "agent_id", agent.ID, "err", err)
	}
	r.publish(ctx, flow.ID, domain.EventSubtaskEscalated, "Runtime escalated a blocked subtask to the operator bridge", map[string]any{
		"agent_role":     role,
		"failed_task":    failedTask.Name,
		"failed_subtask": failedSubtask.Name,
	})
	return nil
}

func (r *Runtime) executeAction(ctx context.Context, flow domain.Flow, taskID, subtaskID, role, taskName, subtaskName string, spec ActionSpec) functions.CallResult {
	input := cloneStringMap(spec.Input)
	for key, value := range r.buildContext(ctx, flow, role, taskName, subtaskName) {
		if _, exists := input[key]; !exists {
			input[key] = value
		}
	}

	r.publish(ctx, flow.ID, domain.EventAgentContext, "Runtime prepared execution context for an action", map[string]any{
		"agent_role":    role,
		"task_name":     taskName,
		"subtask_name":  subtaskName,
		"function_name": spec.FunctionName,
	})

	if r.execute == nil {
		return functions.CallResult{
			Profile: "control",
			Runtime: "go-agent-runtime",
			Output:  map[string]string{"summary": "Action executor not configured."},
			Err:     errors.New("action executor not configured"),
		}
	}
	return r.execute(ctx, ActionInvocation{
		FlowID:       flow.ID,
		TaskID:       taskID,
		SubtaskID:    subtaskID,
		Role:         role,
		FunctionName: spec.FunctionName,
		Input:        input,
	})
}

func (r *Runtime) buildContext(ctx context.Context, flow domain.Flow, role, taskName, subtaskName string) map[string]string {
	memorySummary := memorySummaryFromFlow(ctx, r.repo.SearchMemories, flow.ID)

	return map[string]string{
		"target":              flow.Target,
		"objective":           flow.Objective,
		"task_name":           taskName,
		"subtask_name":        subtaskName,
		"agent_prompt":        r.prompts.Render(role, PromptContext{Flow: flow, TaskName: taskName, SubtaskName: subtaskName, MemorySummary: memorySummary, AvailableFunctions: r.defs}),
		"memory_summary":      memorySummary,
		"available_functions": joinFunctionNames(r.defs),
	}
}

func repeatedToolCallExceeded(observations []ActionObservation, step ActionSpec) bool {
	if step.FunctionName == "done" || step.FunctionName == "ask" {
		return false
	}
	repeatCount := 0
	targetFingerprint := actionFingerprint(step.FunctionName, step.Input)
	for idx := len(observations) - 1; idx >= 0; idx-- {
		if actionFingerprint(observations[idx].FunctionName, observations[idx].Input) != targetFingerprint {
			break
		}
		repeatCount++
	}
	return repeatCount >= 2
}

func actionFingerprint(functionName string, input map[string]string) string {
	keys := make([]string, 0, len(input))
	for key := range input {
		if strings.HasPrefix(key, "_nyx_policy_") {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString(strings.TrimSpace(functionName))
	for _, key := range keys {
		b.WriteByte('|')
		b.WriteString(key)
		b.WriteByte('=')
		b.WriteString(strings.TrimSpace(input[key]))
	}
	return b.String()
}

func joinFunctionNames(defs []domain.FunctionDef) string {
	names := make([]string, 0, len(defs))
	for _, def := range defs {
		names = append(names, def.Name)
	}
	return strings.Join(names, ", ")
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
