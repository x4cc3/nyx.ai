package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"nyx/internal/agentruntime"
)

func (c *Client) NextAction(ctx context.Context, req agentruntime.ActionPolicyRequest) (agentruntime.ActionDecision, agentruntime.ActionPolicyMetadata, error) {
	if strings.TrimSpace(c.apiKey) == "" {
		return agentruntime.ActionDecision{}, agentruntime.ActionPolicyMetadata{}, fmt.Errorf("missing OPENAI_API_KEY")
	}

	payload := map[string]any{
		"model":             c.model,
		"store":             false,
		"max_output_tokens": minInt(c.maxOutputTokens, 4000),
		"reasoning": map[string]any{
			"effort": c.reasoningEffort,
		},
		"text": map[string]any{
			"format": map[string]any{
				"type":        "json_schema",
				"name":        "nyx_action_decision",
				"description": "The next safe NYX function call for the active pentest subtask.",
				"strict":      true,
				"schema":      actionDecisionSchema(functionNamesFromDefs(req.AvailableFunctions)),
			},
		},
		"input": []map[string]any{
			{
				"role": "system",
				"content": []map[string]any{
					{
						"type": "input_text",
						"text": actionSystemPrompt(),
					},
				},
			},
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "input_text",
						"text": renderActionUserPrompt(req),
					},
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return agentruntime.ActionDecision{}, agentruntime.ActionPolicyMetadata{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/responses", bytes.NewReader(body))
	if err != nil {
		return agentruntime.ActionDecision{}, agentruntime.ActionPolicyMetadata{}, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return agentruntime.ActionDecision{}, agentruntime.ActionPolicyMetadata{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return agentruntime.ActionDecision{}, agentruntime.ActionPolicyMetadata{}, err
	}
	if resp.StatusCode >= 300 {
		return agentruntime.ActionDecision{}, agentruntime.ActionPolicyMetadata{}, fmt.Errorf("openai responses api failed with %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed responsesCreateResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return agentruntime.ActionDecision{}, agentruntime.ActionPolicyMetadata{}, err
	}
	text := parsed.outputText()
	if strings.TrimSpace(text) == "" {
		return agentruntime.ActionDecision{}, agentruntime.ActionPolicyMetadata{}, fmt.Errorf("openai response did not include action decision text")
	}

	var decision agentruntime.ActionDecision
	if err := json.Unmarshal([]byte(text), &decision); err != nil {
		return agentruntime.ActionDecision{}, agentruntime.ActionPolicyMetadata{}, fmt.Errorf("decode action decision: %w", err)
	}

	return decision, agentruntime.ActionPolicyMetadata{
		Model:            strings.TrimSpace(parsed.Model),
		ResponseID:       strings.TrimSpace(parsed.ID),
		ReasoningSummary: parsed.reasoningSummary(),
	}, nil
}

func actionSystemPrompt() string {
	return strings.TrimSpace(`
You are the NYX autonomous action policy for controlled pentesting engagements.

Pick exactly one next NYX function call.
Use only the provided NYX functions.
Prefer browser or search actions before terminal execution unless terminal validation is clearly the best next step.
When you choose terminal, provide a concrete command in input.command.
Commands must stay minimal, safe, and scoped to the current subtask.
Use done only when the subtask objective is satisfied with current evidence.
Use ask only when operator input is truly required.
Never invent credentials, data, or tool names.
`)
}

func renderActionUserPrompt(req agentruntime.ActionPolicyRequest) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Flow: %s\n", req.Flow.Name)
	fmt.Fprintf(&b, "Target: %s\n", req.Flow.Target)
	fmt.Fprintf(&b, "Objective: %s\n", req.Flow.Objective)
	fmt.Fprintf(&b, "Task: %s\n", req.Task.Name)
	fmt.Fprintf(&b, "Task description: %s\n", req.Task.Description)
	fmt.Fprintf(&b, "Subtask: %s\n", req.Subtask.Name)
	fmt.Fprintf(&b, "Subtask description: %s\n", req.Subtask.Description)
	fmt.Fprintf(&b, "Attempt: %d\n", req.Attempt)
	fmt.Fprintf(&b, "Turn: %d\n", req.Turn)
	if strings.TrimSpace(req.MemorySummary) != "" {
		fmt.Fprintf(&b, "Memory summary: %s\n", req.MemorySummary)
	}
	if len(req.PlannedSteps) > 0 {
		b.WriteString("\nPlanned step hints:\n")
		for _, step := range req.PlannedSteps {
			fmt.Fprintf(&b, "- %s %v\n", step.FunctionName, step.Input)
		}
	}
	if len(req.Observations) > 0 {
		b.WriteString("\nObserved history:\n")
		for idx, item := range req.Observations {
			fmt.Fprintf(&b, "%d. function=%s input=%v output=%v error=%s\n", idx+1, item.FunctionName, item.Input, item.Output, item.Error)
		}
	}
	b.WriteString("\nAvailable functions:\n")
	for _, def := range req.AvailableFunctions {
		fmt.Fprintf(&b, "- %s (%s): %s\n", def.Name, def.Profile, def.Description)
	}
	b.WriteString("\nReturn the next function call as JSON. Include only the input keys needed for the selected function.\n")
	return b.String()
}

func actionDecisionSchema(functionNames []string) map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"function_name", "input"},
		"properties": map[string]any{
			"function_name": map[string]any{
				"type": "string",
				"enum": functionNames,
			},
			"input": map[string]any{
				"type":                 "object",
				"additionalProperties": map[string]any{"type": "string"},
			},
		},
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
