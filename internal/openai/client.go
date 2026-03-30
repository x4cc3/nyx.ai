package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"nyx/internal/agentruntime"
	"nyx/internal/domain"
)

type ClientConfig struct {
	APIKey          string
	BaseURL         string
	Model           string
	ReasoningEffort string
	MaxOutputTokens int
	HTTPClient      *http.Client
}

type Client struct {
	apiKey          string
	baseURL         string
	model           string
	reasoningEffort string
	maxOutputTokens int
	httpClient      *http.Client
}

func NewClient(cfg ClientConfig) *Client {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	if strings.TrimSpace(cfg.Model) == "" {
		cfg.Model = "gpt-5.1-codex-mini"
	}
	if strings.TrimSpace(cfg.ReasoningEffort) == "" {
		cfg.ReasoningEffort = "high"
	}
	if cfg.MaxOutputTokens < 256 {
		cfg.MaxOutputTokens = 8000
	}
	return &Client{
		apiKey:          cfg.APIKey,
		baseURL:         baseURL,
		model:           cfg.Model,
		reasoningEffort: cfg.ReasoningEffort,
		maxOutputTokens: cfg.MaxOutputTokens,
		httpClient:      client,
	}
}

func (c *Client) Plan(ctx context.Context, req agentruntime.PlannerRequest) (agentruntime.FlowPlan, agentruntime.PlannerMetadata, error) {
	if strings.TrimSpace(c.apiKey) == "" {
		return agentruntime.FlowPlan{}, agentruntime.PlannerMetadata{}, fmt.Errorf("missing OPENAI_API_KEY")
	}

	payload := map[string]any{
		"model":             c.model,
		"store":             false,
		"max_output_tokens": c.maxOutputTokens,
		"reasoning": map[string]any{
			"effort": c.reasoningEffort,
		},
		"text": map[string]any{
			"format": map[string]any{
				"type":        "json_schema",
				"name":        "nyx_flow_plan",
				"description": "A minimal safe NYX execution plan for pentesting with available tools.",
				"strict":      true,
				"schema":      planSchema(functionNamesFromDefs(req.AvailableFunctions)),
			},
		},
		"input": []map[string]any{
			{
				"role": "system",
				"content": []map[string]any{
					{
						"type": "input_text",
						"text": systemPrompt(),
					},
				},
			},
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "input_text",
						"text": renderUserPrompt(req),
					},
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return agentruntime.FlowPlan{}, agentruntime.PlannerMetadata{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/responses", bytes.NewReader(body))
	if err != nil {
		return agentruntime.FlowPlan{}, agentruntime.PlannerMetadata{}, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return agentruntime.FlowPlan{}, agentruntime.PlannerMetadata{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return agentruntime.FlowPlan{}, agentruntime.PlannerMetadata{}, err
	}
	if resp.StatusCode >= 300 {
		return agentruntime.FlowPlan{}, agentruntime.PlannerMetadata{}, fmt.Errorf("openai responses api failed with %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed responsesCreateResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return agentruntime.FlowPlan{}, agentruntime.PlannerMetadata{}, err
	}

	text := parsed.outputText()
	if strings.TrimSpace(text) == "" {
		return agentruntime.FlowPlan{}, agentruntime.PlannerMetadata{}, fmt.Errorf("openai response did not include structured plan text")
	}

	var plan agentruntime.FlowPlan
	if err := json.Unmarshal([]byte(text), &plan); err != nil {
		return agentruntime.FlowPlan{}, agentruntime.PlannerMetadata{}, fmt.Errorf("decode structured plan: %w", err)
	}

	return plan, agentruntime.PlannerMetadata{
		Model:            strings.TrimSpace(parsed.Model),
		ResponseID:       strings.TrimSpace(parsed.ID),
		ReasoningSummary: parsed.reasoningSummary(),
	}, nil
}

type responsesCreateResponse struct {
	ID     string `json:"id"`
	Model  string `json:"model"`
	Output []struct {
		Type    string `json:"type"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
	Reasoning struct {
		Summary []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"summary"`
	} `json:"reasoning"`
}

func (r responsesCreateResponse) outputText() string {
	for _, item := range r.Output {
		for _, content := range item.Content {
			if strings.TrimSpace(content.Text) != "" {
				return content.Text
			}
		}
	}
	return ""
}

func (r responsesCreateResponse) reasoningSummary() string {
	parts := make([]string, 0, len(r.Reasoning.Summary))
	for _, item := range r.Reasoning.Summary {
		if strings.TrimSpace(item.Text) == "" {
			continue
		}
		parts = append(parts, strings.TrimSpace(item.Text))
	}
	return strings.Join(parts, " ")
}

func systemPrompt() string {
	return strings.TrimSpace(`
You are the NYX agent planner for controlled pentesting engagements.

Create a compact, high-signal execution plan using only the provided NYX functions.
Prefer browser or search-based discovery before terminal actions when the target looks like a web app.
Keep the plan minimal, reviewable, and safe for an isolated execution runtime.
Only choose function names from the provided registry.
Do not invent tools, shells, credentials, or access.
Use terminal and file sparingly and only when they materially advance the objective.
Every subtask must include at least one concrete tool step.
Escalate to planner or operator roles when uncertainty or repeated failure is likely.
`)
}

func renderUserPrompt(req agentruntime.PlannerRequest) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Flow name: %s\n", req.Flow.Name)
	fmt.Fprintf(&b, "Target: %s\n", req.Flow.Target)
	fmt.Fprintf(&b, "Objective: %s\n", req.Flow.Objective)
	if strings.TrimSpace(req.MemorySummary) != "" {
		fmt.Fprintf(&b, "Prior memory: %s\n", req.MemorySummary)
	}
	b.WriteString("\nAvailable functions:\n")
	for _, def := range req.AvailableFunctions {
		fmt.Fprintf(&b, "- %s (%s): %s\n", def.Name, def.Profile, def.Description)
	}
	b.WriteString("\nOutput a JSON plan with 1-4 tasks. Keep each task tightly scoped, use 1-3 subtasks, and prefer concise input fields.\n")
	return b.String()
}

func planSchema(functionNames []string) map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"tasks"},
		"properties": map[string]any{
			"tasks": map[string]any{
				"type":     "array",
				"minItems": 1,
				"maxItems": 4,
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"required":             []string{"role", "name", "description", "subtasks"},
					"properties": map[string]any{
						"role": map[string]any{
							"type": "string",
							"enum": []string{"planner", "researcher", "executor", "operator"},
						},
						"name":        map[string]any{"type": "string"},
						"description": map[string]any{"type": "string"},
						"subtasks": map[string]any{
							"type":     "array",
							"minItems": 1,
							"maxItems": 4,
							"items": map[string]any{
								"type":                 "object",
								"additionalProperties": false,
								"required":             []string{"role", "name", "description", "max_attempts", "escalate_to", "escalate_text", "steps"},
								"properties": map[string]any{
									"role": map[string]any{
										"type": "string",
										"enum": []string{"planner", "researcher", "executor", "operator"},
									},
									"name":         map[string]any{"type": "string"},
									"description":  map[string]any{"type": "string"},
									"max_attempts": map[string]any{"type": "integer", "minimum": 1, "maximum": 3},
									"escalate_to": map[string]any{
										"type": "string",
										"enum": []string{"planner", "researcher", "executor", "operator"},
									},
									"escalate_text": map[string]any{"type": "string"},
									"steps": map[string]any{
										"type":     "array",
										"minItems": 1,
										"maxItems": 4,
										"items": map[string]any{
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
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func functionNamesFromDefs(defs []domain.FunctionDef) []string {
	if len(defs) == 0 {
		return []string{"ask"}
	}
	names := make([]string, 0, len(defs))
	seen := make(map[string]struct{}, len(defs))
	for _, def := range defs {
		name := strings.TrimSpace(def.Name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	if len(names) == 0 {
		return []string{"ask"}
	}
	return names
}

var _ agentruntime.Planner = (*Client)(nil)
var _ = domain.Flow{}
