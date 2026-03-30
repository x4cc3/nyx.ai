package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"nyx/internal/agentruntime"
	"nyx/internal/config"
	"nyx/internal/domain"
	"nyx/internal/functions"
	"nyx/internal/queue"
)

func (o *Orchestrator) executeRuntimeAction(ctx context.Context, invocation agentruntime.ActionInvocation) functions.CallResult {
	return o.executeFunction(ctx, invocation.FlowID, invocation.TaskID, invocation.SubtaskID, invocation.Role, invocation.FunctionName, invocation.Input)
}

func (o *Orchestrator) executeFunction(ctx context.Context, flowID, taskID, subtaskID, role, functionName string, input map[string]string) functions.CallResult {
	select {
	case <-ctx.Done():
		return functions.CallResult{
			Profile: "control",
			Runtime: "go-orchestrator",
			Output: map[string]string{
				"summary": "Action execution cancelled before dispatch.",
				"error":   ctx.Err().Error(),
			},
			Err: ctx.Err(),
		}
	default:
	}

	action, err := o.repo.CreateAction(ctx, flowID, taskID, subtaskID, role, functionName, "ephemeral", input)
	if err != nil {
		o.logger.Warn("failed to create action", "flow_id", flowID, "function", functionName, "err", err)
	}
	exec, execErr := o.repo.CreateExecution(ctx, flowID, action.ID, functionName, "go-runtime", initialExecutionMetadata(functionName, role, action.ExecutionMode, input))
	if execErr != nil {
		o.logger.Warn("failed to create execution", "flow_id", flowID, "action_id", action.ID, "err", execErr)
	}
	if approvalResult, approvalTriggered := o.maybeRequireRiskApproval(ctx, flowID, action.ID, functionName, input); approvalTriggered {
		actionStatus := domain.StatusFailed
		execStatus := domain.StatusFailed
		eventType := domain.EventActionFailed
		eventMessage := "Function call failed before execution"
		artifactKind := domain.ArtifactKindLog
		if functions.ApprovalRequired(approvalResult) {
			actionStatus = domain.StatusPending
			execStatus = domain.StatusPending
			eventType = domain.EventActionApprovalRequested
			eventMessage = "Function call paused for operator approval"
			artifactKind = domain.ArtifactKindApproval
		}
		action, err = o.repo.CompleteAction(ctx, action.ID, actionStatus, approvalResult.Output)
		if err != nil {
			o.logger.Warn("failed to complete action", "action_id", action.ID, "err", err)
		}
		execMetadata := finalExecutionMetadata(functionName, role, action.ExecutionMode, input, approvalResult)
		profile := "approval"
		if !functions.ApprovalRequired(approvalResult) {
			profile = blankToDefault(approvalResult.Profile, functionName)
		}
		if err := o.repo.CompleteExecution(ctx, exec.ID, execStatus, profile, blankToDefault(approvalResult.Runtime, "go-orchestrator"), execMetadata); err != nil {
			o.logger.Warn("failed to complete execution", "execution_id", exec.ID, "err", err)
		}
		if _, err := o.repo.AddArtifact(ctx, flowID, action.ID, artifactKind, functionName+"-"+artifactKind, blankToDefault(approvalResult.Output["summary"], eventMessage), execMetadata); err != nil {
			o.logger.Warn("failed to add artifact", "flow_id", flowID, "action_id", action.ID, "err", err)
		}
		o.publish(ctx, flowID, eventType, eventMessage, map[string]any{
			"action_id":       action.ID,
			"function_name":   functionName,
			"agent_role":      role,
			"approval_id":     approvalResult.Output["approval_id"],
			"risk_categories": approvalResult.Output["risk_categories"],
		})
		o.logger.Info("action blocked before execution",
			"flow_id", flowID,
			"action_id", action.ID,
			"function", functionName,
			"approval_id", approvalResult.Output["approval_id"],
			"risk_categories", approvalResult.Output["risk_categories"],
			"error", approvalResult.Err,
		)
		return approvalResult
	}
	o.metrics.IncCounter("nyx_orchestrator_actions_total", map[string]string{
		"function": functionName,
		"mode":     o.transport.Mode(),
	}, 1)
	o.publish(ctx, flowID, domain.EventActionStarted, "Function call started", map[string]any{
		"action_id":     action.ID,
		"function_name": functionName,
		"agent_role":    role,
	})

	result := functions.CallResult{}
	if o.transport != nil && o.transport.Mode() == config.TransportJetstream {
		waitCtx, cancel := context.WithTimeout(ctx, o.actionWait)
		defer cancel()

		queueResult, err := o.transport.DispatchAction(waitCtx, queue.ActionRequestMessage{
			FlowID:        flowID,
			TaskID:        taskID,
			SubtaskID:     subtaskID,
			ActionID:      action.ID,
			AgentRole:     role,
			FunctionName:  functionName,
			ExecutionMode: action.ExecutionMode,
			Input:         input,
		})
		result = functions.CallResult{
			Output:  queueResult.Output,
			Profile: queueResult.Profile,
			Runtime: queueResult.Runtime,
		}
		if queueResult.Error != "" {
			result.Err = errors.New(queueResult.Error)
		}
		if err != nil {
			result.Err = err
			if result.Output == nil {
				result.Output = map[string]string{}
			}
			result.Output["summary"] = "Action transport failed before executor completion."
		}
	} else {
		result = o.gateway.Call(ctx, flowID, action.ID, functionName, input)
	}
	actionStatus := domain.StatusCompleted
	execStatus := domain.StatusCompleted
	eventType := domain.EventActionCompleted
	eventMessage := "Function call completed"
	if result.Err != nil {
		actionStatus = domain.StatusFailed
		execStatus = domain.StatusFailed
		o.metrics.IncCounter("nyx_orchestrator_action_failures_total", map[string]string{"function": functionName}, 1)
		result.Output["error"] = result.Err.Error()
		eventType = domain.EventActionFailed
		eventMessage = "Function call failed"
	}
	action, err = o.repo.CompleteAction(ctx, action.ID, actionStatus, result.Output)
	if err != nil {
		o.logger.Warn("failed to complete action", "action_id", action.ID, "err", err)
	}
	execMetadata := finalExecutionMetadata(functionName, role, action.ExecutionMode, input, result)
	if err := o.repo.CompleteExecution(ctx, exec.ID, execStatus, blankToDefault(result.Profile, functionName), blankToDefault(result.Runtime, "go-runtime"), execMetadata); err != nil {
		o.logger.Warn("failed to complete execution", "execution_id", exec.ID, "err", err)
	}

	if _, err := o.repo.AddArtifact(ctx, flowID, action.ID, domain.ArtifactKindLog, functionName+"-output", blankToDefault(result.Output["summary"], eventMessage), execMetadata); err != nil {
		o.logger.Warn("failed to add artifact", "flow_id", flowID, "action_id", action.ID, "kind", "log", "err", err)
	}
	if _, err := o.repo.AddArtifact(ctx, flowID, action.ID, domain.ArtifactKindExecution, functionName+"-trace", renderExecutionTrace(blankToDefault(result.Profile, functionName), blankToDefault(result.Runtime, "go-runtime"), execMetadata), execMetadata); err != nil {
		o.logger.Warn("failed to add artifact", "flow_id", flowID, "action_id", action.ID, "kind", "execution", "err", err)
	}
	if stdout := strings.TrimSpace(result.Output["stdout"]); stdout != "" {
		if _, err := o.repo.AddArtifact(ctx, flowID, action.ID, domain.ArtifactKindStdout, functionName+"-stdout", stdout, execMetadata); err != nil {
			o.logger.Warn("failed to add artifact", "flow_id", flowID, "action_id", action.ID, "kind", "stdout", "err", err)
		}
	}
	if stderr := strings.TrimSpace(result.Output["stderr"]); stderr != "" {
		if _, err := o.repo.AddArtifact(ctx, flowID, action.ID, domain.ArtifactKindStderr, functionName+"-stderr", stderr, execMetadata); err != nil {
			o.logger.Warn("failed to add artifact", "flow_id", flowID, "action_id", action.ID, "kind", "stderr", "err", err)
		}
	}
	if path := strings.TrimSpace(result.Output["screenshot_path"]); path != "" {
		if _, err := o.repo.AddArtifact(ctx, flowID, action.ID, domain.ArtifactKindScreenshot, functionName+"-screenshot", path, execMetadata); err != nil {
			o.logger.Warn("failed to add artifact", "flow_id", flowID, "action_id", action.ID, "kind", "screenshot", "err", err)
		}
	}
	if path := strings.TrimSpace(result.Output["snapshot_path"]); path != "" {
		if _, err := o.repo.AddArtifact(ctx, flowID, action.ID, domain.ArtifactKindSnapshot, functionName+"-snapshot", path, execMetadata); err != nil {
			o.logger.Warn("failed to add artifact", "flow_id", flowID, "action_id", action.ID, "kind", "snapshot", "err", err)
		}
	}
	if o.memory != nil {
		o.memory.StoreActionResult(ctx, flowID, action.ID, role, functionName, input, result.Output)
	}

	o.publish(ctx, flowID, eventType, eventMessage, map[string]any{
		"action_id":     action.ID,
		"function_name": functionName,
		"output":        result.Output,
	})
	o.logger.Info("action completed",
		"flow_id", flowID,
		"action_id", action.ID,
		"function", functionName,
		"status", actionStatus,
		"runtime", result.Runtime,
	)
	return result
}

func initialExecutionMetadata(functionName, role, executionMode string, input map[string]string) map[string]string {
	metadata := map[string]string{
		"function_name":  strings.TrimSpace(functionName),
		"agent_role":     strings.TrimSpace(role),
		"execution_mode": strings.TrimSpace(executionMode),
	}
	copyExecutionField(metadata, input, "command")
	copyExecutionField(metadata, input, "goal")
	copyExecutionField(metadata, input, "target")
	copyExecutionField(metadata, input, "url")
	copyExecutionField(metadata, input, "path")
	copyExecutionField(metadata, input, "toolset")
	return metadata
}

func finalExecutionMetadata(functionName, role, executionMode string, input map[string]string, result functions.CallResult) map[string]string {
	metadata := initialExecutionMetadata(functionName, role, executionMode, input)
	for _, key := range []string{
		"workspace",
		"image",
		"toolset",
		"network_mode",
		"network_name",
		"net_raw",
		"exit_code",
		"duration_ms",
		"attempts",
		"final_url",
		"status_code",
		"mode",
		"screenshot_path",
		"snapshot_path",
	} {
		copyExecutionField(metadata, result.Output, key)
	}
	if trimmed := strings.TrimSpace(result.Profile); trimmed != "" {
		metadata["profile"] = trimmed
	}
	if trimmed := strings.TrimSpace(result.Runtime); trimmed != "" {
		metadata["runtime"] = trimmed
	}
	if evidence := evidencePaths(metadata); evidence != "" {
		metadata["evidence_paths"] = evidence
	}
	return metadata
}

func copyExecutionField(dst, src map[string]string, key string) {
	if dst == nil || src == nil {
		return
	}
	if value := strings.TrimSpace(src[key]); value != "" {
		dst[key] = value
	}
}

func evidencePaths(metadata map[string]string) string {
	paths := make([]string, 0, 3)
	for _, key := range []string{"screenshot_path", "snapshot_path", "workspace"} {
		if value := strings.TrimSpace(metadata[key]); value != "" {
			paths = append(paths, value)
		}
	}
	return strings.Join(paths, ", ")
}

func renderExecutionTrace(profile, runtime string, metadata map[string]string) string {
	lines := []string{
		fmt.Sprintf("profile: %s", blankToDefault(profile, "unknown")),
		fmt.Sprintf("runtime: %s", blankToDefault(runtime, "unknown")),
	}
	keys := make([]string, 0, len(metadata))
	for key, value := range metadata {
		if strings.TrimSpace(value) == "" {
			continue
		}
		if key == "profile" || key == "runtime" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		lines = append(lines, fmt.Sprintf("%s: %s", key, metadata[key]))
	}
	return strings.Join(lines, "\n")
}

func blankToDefault(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return strings.TrimSpace(fallback)
	}
	return value
}
