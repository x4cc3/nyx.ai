package agentruntime

import (
	"context"
	"errors"
	"strings"
	"testing"

	"nyx/internal/domain"
	"nyx/internal/functions"
	"nyx/internal/store"
)

type recordedEvent struct {
	FlowID  string
	Type    string
	Message string
	Payload map[string]any
}

type stubExecutor struct {
	calls    []ActionInvocation
	failures map[string]int
}

type stubPlanner struct {
	plan     FlowPlan
	metadata PlannerMetadata
	err      error
	calls    int
}

type stubActionPolicy struct {
	decisions []ActionDecision
	metadata  ActionPolicyMetadata
	err       error
	calls     int
	requests  []ActionPolicyRequest
}

func (s *stubPlanner) Plan(_ context.Context, _ PlannerRequest) (FlowPlan, PlannerMetadata, error) {
	s.calls++
	return s.plan, s.metadata, s.err
}

func (s *stubActionPolicy) NextAction(_ context.Context, req ActionPolicyRequest) (ActionDecision, ActionPolicyMetadata, error) {
	s.calls++
	s.requests = append(s.requests, req)
	if s.err != nil {
		return ActionDecision{}, s.metadata, s.err
	}
	index := s.calls - 1
	if index >= len(s.decisions) {
		index = len(s.decisions) - 1
	}
	if index < 0 {
		return ActionDecision{}, s.metadata, errors.New("no policy decision configured")
	}
	return s.decisions[index], s.metadata, nil
}

func (s *stubExecutor) Execute(_ context.Context, req ActionInvocation) functions.CallResult {
	s.calls = append(s.calls, req)
	key := req.Role + ":" + req.FunctionName
	if remaining := s.failures[key]; remaining > 0 {
		s.failures[key] = remaining - 1
		return functions.CallResult{
			Profile: req.FunctionName,
			Runtime: "stub-runtime",
			Output: map[string]string{
				"summary": "stub failure",
			},
			Err: errors.New("stubbed action failure"),
		}
	}

	return functions.CallResult{
		Profile: req.FunctionName,
		Runtime: "stub-runtime",
		Output: map[string]string{
			"summary": "stub success",
		},
	}
}

func TestRuntimeRunFlowCompletesWithAgentPlan(t *testing.T) {
	repo := store.NewMemoryStore()
	flow, err := repo.CreateFlow(context.Background(), domain.CreateFlowInput{
		Name:      "Acme external assessment",
		Target:    "https://app.example.com",
		Objective: "Map the target and validate the agent runtime.",
	})
	if err != nil {
		t.Fatalf("create flow: %v", err)
	}
	flow, err = repo.UpdateFlowStatus(context.Background(), flow.ID, domain.StatusRunning)
	if err != nil {
		t.Fatalf("set flow running: %v", err)
	}

	exec := &stubExecutor{failures: map[string]int{}}
	events := make([]recordedEvent, 0)
	runtime := New(repo, []domain.FunctionDef{
		{Name: "search_web"},
		{Name: "search_deep"},
		{Name: "search_code"},
		{Name: "search_exploits"},
		{Name: "browser_markdown"},
		{Name: "browser_links"},
		{Name: "browser_screenshot"},
		{Name: "search_memory"},
		{Name: "terminal_exec", RequiresPentestImage: true},
		{Name: "file_write"},
		{Name: "done"},
		{Name: "ask"},
	}, exec.Execute, func(_ context.Context, flowID, eventType, message string, payload map[string]any) {
		events = append(events, recordedEvent{FlowID: flowID, Type: eventType, Message: message, Payload: payload})
	})

	if err := runtime.RunFlow(context.Background(), flow); err != nil {
		t.Fatalf("run flow: %v", err)
	}

	updatedFlow, err := repo.GetFlow(context.Background(), flow.ID)
	if err != nil {
		t.Fatalf("get flow: %v", err)
	}
	if updatedFlow.Status != domain.StatusCompleted {
		t.Fatalf("expected completed flow, got %s", updatedFlow.Status)
	}

	detail, err := repo.FlowDetail(context.Background(), flow.ID)
	if err != nil {
		t.Fatalf("flow detail: %v", err)
	}
	if len(detail.Tasks) != 3 {
		t.Fatalf("expected 3 runtime tasks, got %d", len(detail.Tasks))
	}
	if len(detail.Agents) != 3 {
		t.Fatalf("expected 3 agents, got %d", len(detail.Agents))
	}
	for _, agent := range detail.Agents {
		if agent.Status != domain.StatusCompleted {
			t.Fatalf("expected completed agent, got %s", agent.Status)
		}
	}

	var sawPrompt bool
	var sawWorkspaceNote bool
	var sawPentestToolset bool
	var sawExploitResearch bool
	var sawRichTerminalMetadata bool
	var sawTargetMemoryNamespace bool
	var sawExploitMemoryNamespace bool
	for _, call := range exec.calls {
		if call.Input["agent_prompt"] != "" && call.Input["available_functions"] != "" {
			sawPrompt = true
		}
		if call.FunctionName == "search_memory" && call.Input["namespace"] == domain.MemoryNamespaceTargetObservations {
			sawTargetMemoryNamespace = true
		}
		if call.FunctionName == "search_memory" && call.Input["namespace"] == domain.MemoryNamespaceExploitReferences {
			sawExploitMemoryNamespace = true
		}
		if call.FunctionName == "terminal_exec" && call.Input["toolset"] == "pentest" {
			sawPentestToolset = true
		}
		if call.FunctionName == "search_exploits" {
			sawExploitResearch = true
		}
		if call.FunctionName == "terminal_exec" &&
			call.Input["network_required"] == "true" &&
			call.Input["evidence_expected"] != "" &&
			call.Input["preferred_output"] == "stdout" &&
			call.Input["command"] != "" {
			sawRichTerminalMetadata = true
		}
		if call.FunctionName == "file_write" && strings.Contains(call.Input["content"], "Runtime: supervised agent loop") {
			sawWorkspaceNote = true
		}
	}
	if !sawPrompt {
		t.Fatal("expected execution context to include an agent prompt and available functions")
	}
	if !sawWorkspaceNote {
		t.Fatal("expected executor workspace note action to be scheduled")
	}
	if !sawPentestToolset {
		t.Fatal("expected terminal actions to default to the pentest toolset")
	}
	if !sawExploitResearch {
		t.Fatal("expected exploit research action to be scheduled")
	}
	if !sawTargetMemoryNamespace {
		t.Fatal("expected runtime memory lookups to default to target_observations")
	}
	if !sawExploitMemoryNamespace {
		t.Fatal("expected exploit research flow to retrieve exploit_references from memory")
	}
	if !sawRichTerminalMetadata {
		t.Fatal("expected terminal validation to include richer execution metadata")
	}

	if !containsEvent(events, "runtime.plan.ready") {
		t.Fatal("expected runtime.plan.ready event")
	}
}

func TestRuntimeRetriesAndEscalatesBlockedSubtasks(t *testing.T) {
	repo := store.NewMemoryStore()
	flow, err := repo.CreateFlow(context.Background(), domain.CreateFlowInput{
		Name:      "Blocked assessment",
		Target:    "https://blocked.example.com",
		Objective: "Exercise runtime escalation rules.",
	})
	if err != nil {
		t.Fatalf("create flow: %v", err)
	}
	flow, err = repo.UpdateFlowStatus(context.Background(), flow.ID, domain.StatusRunning)
	if err != nil {
		t.Fatalf("set flow running: %v", err)
	}

	exec := &stubExecutor{
		failures: map[string]int{
			"executor:terminal_exec": 2,
		},
	}
	events := make([]recordedEvent, 0)
	runtime := New(repo, []domain.FunctionDef{
		{Name: "search_web"},
		{Name: "search_deep"},
		{Name: "search_code"},
		{Name: "search_exploits"},
		{Name: "browser_markdown"},
		{Name: "browser_links"},
		{Name: "search_memory"},
		{Name: "terminal_exec", RequiresPentestImage: true},
		{Name: "file_write"},
		{Name: "done"},
		{Name: "ask"},
	}, exec.Execute, func(_ context.Context, flowID, eventType, message string, payload map[string]any) {
		events = append(events, recordedEvent{FlowID: flowID, Type: eventType, Message: message, Payload: payload})
	})

	if err := runtime.RunFlow(context.Background(), flow); err != nil {
		t.Fatalf("expected flow to continue after escalation, got %v", err)
	}

	updatedFlow, err := repo.GetFlow(context.Background(), flow.ID)
	if err != nil {
		t.Fatalf("get flow: %v", err)
	}
	if updatedFlow.Status != domain.StatusCompleted {
		t.Fatalf("expected completed flow after escalation, got %s", updatedFlow.Status)
	}

	detail, err := repo.FlowDetail(context.Background(), flow.ID)
	if err != nil {
		t.Fatalf("flow detail: %v", err)
	}
	if len(detail.Findings) == 0 {
		t.Fatal("expected escalation finding to be recorded")
	}
	var sawEscalationTask bool
	for _, task := range detail.Tasks {
		if task.Name == "Escalation review" {
			sawEscalationTask = true
			break
		}
	}
	if !sawEscalationTask {
		t.Fatal("expected escalation review task to be created")
	}

	terminalCalls := 0
	askCalls := 0
	for _, call := range exec.calls {
		if call.Role == "executor" && call.FunctionName == "terminal_exec" {
			terminalCalls++
		}
		if call.FunctionName == "ask" {
			askCalls++
		}
	}
	if terminalCalls != 2 {
		t.Fatalf("expected 2 terminal attempts, got %d", terminalCalls)
	}
	if askCalls != 1 {
		t.Fatalf("expected 1 ask escalation action, got %d", askCalls)
	}

	if !containsEvent(events, "subtask.retry") {
		t.Fatal("expected subtask.retry event")
	}
	if !containsEvent(events, "task.reassigned") {
		t.Fatal("expected task.reassigned event")
	}
}

func TestRuntimeUsesPlannerOutputWhenConfigured(t *testing.T) {
	repo := store.NewMemoryStore()
	flow, err := repo.CreateFlow(context.Background(), domain.CreateFlowInput{
		Name:      "Planner driven assessment",
		Target:    "https://planner.example.com",
		Objective: "Use OpenAI-generated planning output.",
	})
	if err != nil {
		t.Fatalf("create flow: %v", err)
	}
	flow, err = repo.UpdateFlowStatus(context.Background(), flow.ID, domain.StatusRunning)
	if err != nil {
		t.Fatalf("set flow running: %v", err)
	}

	exec := &stubExecutor{failures: map[string]int{}}
	planner := &stubPlanner{
		metadata: PlannerMetadata{Model: "gpt-5.1-codex-mini", ResponseID: "resp_123"},
		plan: FlowPlan{
			Tasks: []TaskSpec{
				{
					Role:        "researcher",
					Name:        "Planner-generated research",
					Description: "Use the browser first.",
					Subtasks: []SubtaskSpec{
						{
							Role:         "researcher",
							Name:         "Browser trace",
							Description:  "Capture the target page.",
							MaxAttempts:  2,
							EscalateTo:   "planner",
							EscalateText: "Need planner support.",
							Steps: []ActionSpec{
								{FunctionName: "browser", Input: map[string]string{"url": "https://planner.example.com"}},
							},
						},
					},
				},
			},
		},
	}
	runtime := New(
		repo,
		[]domain.FunctionDef{{Name: "browser"}, {Name: "ask"}, {Name: "done"}},
		exec.Execute,
		nil,
		WithPlanner(planner),
		WithPromptLibrary(DefaultPromptLibrary("gpt-5.1-codex-mini")),
	)

	if err := runtime.RunFlow(context.Background(), flow); err != nil {
		t.Fatalf("run flow: %v", err)
	}
	if planner.calls != 1 {
		t.Fatalf("expected planner to be called once, got %d", planner.calls)
	}
	detail, err := repo.FlowDetail(context.Background(), flow.ID)
	if err != nil {
		t.Fatalf("flow detail: %v", err)
	}
	var sawPlannerTask bool
	var sawPlanArtifact bool
	for _, task := range detail.Tasks {
		if task.Name == "Planner-generated research" {
			sawPlannerTask = true
		}
	}
	for _, artifact := range detail.Artifacts {
		if artifact.Kind == "plan" && strings.Contains(artifact.Name, "gpt-5.1-codex-mini") {
			sawPlanArtifact = true
		}
	}
	if !sawPlannerTask {
		t.Fatal("expected planner-generated task to be persisted")
	}
	if !sawPlanArtifact {
		t.Fatal("expected planner artifact to be recorded")
	}
}

func TestRuntimeUsesActionPolicyForConcreteNextStep(t *testing.T) {
	repo := store.NewMemoryStore()
	flow, err := repo.CreateFlow(context.Background(), domain.CreateFlowInput{
		Name:      "Autonomous execution",
		Target:    "https://app.example.com",
		Objective: "Derive a concrete validation command from the model.",
	})
	if err != nil {
		t.Fatalf("create flow: %v", err)
	}
	flow, err = repo.UpdateFlowStatus(context.Background(), flow.ID, domain.StatusRunning)
	if err != nil {
		t.Fatalf("set flow running: %v", err)
	}

	exec := &stubExecutor{failures: map[string]int{}}
	policy := &stubActionPolicy{
		metadata: ActionPolicyMetadata{Model: "gpt-5.1-codex-mini", ResponseID: "resp_action_1"},
		decisions: []ActionDecision{
			{FunctionName: "terminal", Input: map[string]string{"command": "printf 'HEAD / HTTP/1.1\\r\\nHost: app.example.com\\r\\n\\r\\n' | nc -w 2 app.example.com 80", "goal": "Probe HTTP response headers safely"}},
			{FunctionName: "done", Input: map[string]string{"summary": "Terminal validation captured a safe baseline response."}},
		},
	}

	runtime := New(repo, []domain.FunctionDef{
		{Name: "search_web"},
		{Name: "browser"},
		{Name: "search_memory"},
		{Name: "terminal"},
		{Name: "file"},
		{Name: "done"},
		{Name: "ask"},
	}, exec.Execute, nil, WithActionPolicy(policy))

	task := TaskSpec{
		Role:        "executor",
		Name:        "Execution",
		Description: "Run a controlled terminal validation.",
		Subtasks: []SubtaskSpec{{
			Role:        "executor",
			Name:        "Probe",
			Description: "Capture a minimal baseline response.",
			MaxAttempts: 1,
			Steps: []ActionSpec{{
				FunctionName: "terminal",
				Input:        map[string]string{"goal": "fallback probe"},
			}},
		}},
	}

	if err := runtime.runTask(context.Background(), flow, task); err != nil {
		t.Fatalf("run task: %v", err)
	}
	if policy.calls != 2 {
		t.Fatalf("expected 2 policy calls, got %d", policy.calls)
	}
	if len(exec.calls) != 2 {
		t.Fatalf("expected 2 executed actions, got %d", len(exec.calls))
	}
	if exec.calls[0].FunctionName != "terminal" {
		t.Fatalf("expected terminal first, got %s", exec.calls[0].FunctionName)
	}
	if !strings.Contains(exec.calls[0].Input["command"], "nc -w 2 app.example.com 80") {
		t.Fatalf("expected concrete command from policy, got %q", exec.calls[0].Input["command"])
	}
	if exec.calls[1].FunctionName != "done" {
		t.Fatalf("expected done second, got %s", exec.calls[1].FunctionName)
	}
}

func TestRuntimeDecomposeWebTargetEmitsPentestReadySteps(t *testing.T) {
	repo := store.NewMemoryStore()
	runtime := New(repo, []domain.FunctionDef{
		{Name: "search_deep"},
		{Name: "search_code"},
		{Name: "search_exploits"},
		{Name: "browser_markdown"},
		{Name: "browser_links"},
		{Name: "browser_screenshot"},
		{Name: "search_memory"},
		{Name: "terminal_exec", RequiresPentestImage: true},
		{Name: "file_write"},
	}, func(context.Context, ActionInvocation) functions.CallResult {
		return functions.CallResult{}
	}, nil)

	plan := runtime.decompose(domain.Flow{
		Name:      "Acme web target",
		Target:    "https://app.example.com",
		Objective: "Check admin paths and likely vulnerability exposure.",
	})

	if len(plan.Tasks) != 2 {
		t.Fatalf("expected 2 planned tasks, got %d", len(plan.Tasks))
	}
	research := plan.Tasks[0]
	if len(research.Subtasks) != 2 {
		t.Fatalf("expected 2 research subtasks, got %d", len(research.Subtasks))
	}
	if research.Subtasks[1].Name != "Exploit reference triage" {
		t.Fatalf("unexpected exploit subtask: %+v", research.Subtasks[1])
	}
	execution := plan.Tasks[1]
	if len(execution.Subtasks) < 2 {
		t.Fatalf("expected execution subtasks, got %+v", execution.Subtasks)
	}
	baseline := execution.Subtasks[0].Steps[0]
	if baseline.FunctionName != "terminal_exec" {
		t.Fatalf("expected terminal_exec baseline step, got %+v", baseline)
	}
	if !strings.Contains(baseline.Input["command"], "httpx -u") {
		t.Fatalf("expected httpx baseline command, got %q", baseline.Input["command"])
	}
	if baseline.Input["network_required"] != "true" || baseline.Input["preferred_output"] != "stdout" {
		t.Fatalf("expected rich baseline metadata, got %+v", baseline.Input)
	}
	targeted := execution.Subtasks[1].Steps
	if len(targeted) == 0 {
		t.Fatal("expected targeted validation steps")
	}
}

func TestRuntimeBlocksRepeatedPolicyToolCalls(t *testing.T) {
	repo := store.NewMemoryStore()
	flow, err := repo.CreateFlow(context.Background(), domain.CreateFlowInput{
		Name:      "Policy loop",
		Target:    "https://app.example.com",
		Objective: "Exercise repeated tool call budgeting.",
	})
	if err != nil {
		t.Fatalf("create flow: %v", err)
	}
	flow, err = repo.UpdateFlowStatus(context.Background(), flow.ID, domain.StatusRunning)
	if err != nil {
		t.Fatalf("set flow running: %v", err)
	}

	exec := &stubExecutor{failures: map[string]int{}}
	policy := &stubActionPolicy{
		decisions: []ActionDecision{
			{FunctionName: "search_web", Input: map[string]string{"query": "same"}},
			{FunctionName: "search_web", Input: map[string]string{"query": "same"}},
			{FunctionName: "search_web", Input: map[string]string{"query": "same"}},
		},
	}
	runtime := New(repo, []domain.FunctionDef{
		{Name: "search_web"},
		{Name: "done"},
		{Name: "ask"},
	}, exec.Execute, nil, WithActionPolicy(policy))

	task := TaskSpec{
		Role:        "researcher",
		Name:        "Research",
		Description: "Loop test",
		Subtasks: []SubtaskSpec{{
			Role:        "researcher",
			Name:        "Repeated search",
			Description: "Budget repeated search calls.",
			MaxAttempts: 1,
			Steps: []ActionSpec{{
				FunctionName: "search_web",
				Input:        map[string]string{"query": "same"},
			}},
		}},
	}

	if err := runtime.runTask(context.Background(), flow, task); err != nil {
		t.Fatalf("expected task to continue after escalation, got %v", err)
	}
	searchCalls := 0
	askCalls := 0
	for _, call := range exec.calls {
		switch call.FunctionName {
		case "search_web":
			searchCalls++
		case "ask":
			askCalls++
		}
	}
	if searchCalls != 2 {
		t.Fatalf("expected only 2 repeated search executions before blocking, got %d", searchCalls)
	}
	if askCalls != 1 {
		t.Fatalf("expected a single escalation ask action, got %d", askCalls)
	}
}

func containsEvent(events []recordedEvent, eventType string) bool {
	for _, item := range events {
		if item.Type == eventType {
			return true
		}
	}
	return false
}
