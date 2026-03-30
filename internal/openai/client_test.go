package openai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"nyx/internal/agentruntime"
	"nyx/internal/domain"
)

func TestClientPlanParsesStructuredResponse(t *testing.T) {
	var authHeader string
	var requestBody map[string]any

	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			authHeader = r.Header.Get("Authorization")
			if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body: io.NopCloser(strings.NewReader(`{
			"id":"resp_123",
			"model":"gpt-5.1-codex-mini",
			"reasoning":{"summary":[{"type":"summary_text","text":"Chose a narrow browser-first plan."}]},
			"output":[
				{
					"type":"message",
					"content":[
						{
							"type":"output_text",
							"text":"{\"tasks\":[{\"role\":\"researcher\",\"name\":\"Surface mapping\",\"description\":\"Inspect the HTTP target and capture evidence.\",\"subtasks\":[{\"role\":\"researcher\",\"name\":\"Initial browser pass\",\"description\":\"Open the target and retain page context.\",\"max_attempts\":2,\"escalate_to\":\"planner\",\"escalate_text\":\"Need planner review if the page is unreachable.\",\"steps\":[{\"function_name\":\"browser\",\"input\":{\"url\":\"https://app.example.com\"}},{\"function_name\":\"search_memory\",\"input\":{\"query\":\"target snapshot\"}}]}]}]}"
						}
					]
				}
			]
		}`)),
				Request: r,
			}, nil
		}),
	}

	planner := NewClient(ClientConfig{
		APIKey:          "test-key",
		BaseURL:         "https://example.test/v1",
		Model:           "gpt-5.1-codex-mini",
		ReasoningEffort: "high",
		MaxOutputTokens: 2048,
		HTTPClient:      client,
	})

	plan, metadata, err := planner.Plan(context.Background(), agentruntime.PlannerRequest{
		Flow: domain.Flow{
			ID:        "flow_1",
			Name:      "Acme external assessment",
			Target:    "https://app.example.com",
			Objective: "Map the target safely.",
		},
		AvailableFunctions: []domain.FunctionDef{
			{Name: "browser", Profile: "browser", Description: "Navigate to a page."},
			{Name: "search_memory", Profile: "memory", Description: "Search memory."},
		},
	})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}

	if authHeader != "Bearer test-key" {
		t.Fatalf("expected bearer auth, got %q", authHeader)
	}
	if len(plan.Tasks) != 1 || plan.Tasks[0].Name != "Surface mapping" {
		t.Fatalf("unexpected plan: %+v", plan)
	}
	if metadata.Model != "gpt-5.1-codex-mini" || metadata.ResponseID != "resp_123" {
		t.Fatalf("unexpected metadata: %+v", metadata)
	}
	if !strings.Contains(metadata.ReasoningSummary, "browser-first") {
		t.Fatalf("expected reasoning summary, got %q", metadata.ReasoningSummary)
	}

	input, ok := requestBody["input"].([]any)
	if !ok || len(input) != 2 {
		t.Fatalf("expected two input messages, got %+v", requestBody["input"])
	}
	schema := requestBody["text"].(map[string]any)["format"].(map[string]any)["schema"].(map[string]any)
	enum := schema["properties"].(map[string]any)["tasks"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)["subtasks"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)["steps"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)["function_name"].(map[string]any)["enum"].([]any)
	if len(enum) != 2 || enum[0] != "browser" || enum[1] != "search_memory" {
		t.Fatalf("unexpected function enum: %+v", enum)
	}
}

func TestClientPlanRequiresAPIKey(t *testing.T) {
	client := NewClient(ClientConfig{})
	_, _, err := client.Plan(context.Background(), agentruntime.PlannerRequest{})
	if err == nil {
		t.Fatal("expected missing api key error")
	}
}

func TestClientNextActionParsesStructuredResponse(t *testing.T) {
	var requestBody map[string]any
	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body: io.NopCloser(strings.NewReader(`{
			"id":"resp_action_123",
			"model":"gpt-5.1-codex-mini",
			"reasoning":{"summary":[{"type":"summary_text","text":"A browser-first trace is the safest next step."}]},
			"output":[{"type":"message","content":[{"type":"output_text","text":"{\"function_name\":\"browser\",\"input\":{\"url\":\"https://app.example.com/login\",\"wait_selector\":\"form\"}}"}]}]
		}`)),
				Request: r,
			}, nil
		}),
	}

	policy := NewClient(ClientConfig{
		APIKey:          "test-key",
		BaseURL:         "https://example.test/v1",
		Model:           "gpt-5.1-codex-mini",
		ReasoningEffort: "high",
		MaxOutputTokens: 2048,
		HTTPClient:      client,
	})

	decision, metadata, err := policy.NextAction(context.Background(), agentruntime.ActionPolicyRequest{
		Flow: domain.Flow{
			ID:        "flow_1",
			Name:      "Acme external assessment",
			Target:    "https://app.example.com",
			Objective: "Map the login workflow safely.",
		},
		Task:    agentruntime.TaskSpec{Name: "Research"},
		Subtask: agentruntime.SubtaskSpec{Name: "Initial page trace"},
		AvailableFunctions: []domain.FunctionDef{
			{Name: "browser", Profile: "browser", Description: "Navigate to a page."},
			{Name: "done", Profile: "control", Description: "Mark the subtask complete."},
		},
	})
	if err != nil {
		t.Fatalf("next action: %v", err)
	}
	if decision.FunctionName != "browser" {
		t.Fatalf("unexpected function: %s", decision.FunctionName)
	}
	if decision.Input["url"] != "https://app.example.com/login" {
		t.Fatalf("unexpected url: %s", decision.Input["url"])
	}
	if metadata.ResponseID != "resp_action_123" {
		t.Fatalf("unexpected metadata: %+v", metadata)
	}
	schema := requestBody["text"].(map[string]any)["format"].(map[string]any)["schema"].(map[string]any)
	enum := schema["properties"].(map[string]any)["function_name"].(map[string]any)["enum"].([]any)
	if len(enum) != 2 || enum[0] != "browser" || enum[1] != "done" {
		t.Fatalf("unexpected function enum: %+v", enum)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
