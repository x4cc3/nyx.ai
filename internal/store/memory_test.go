package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"nyx/internal/domain"
)

func ctx() context.Context { return context.Background() }

func newInput(name string) domain.CreateFlowInput {
	return domain.CreateFlowInput{Name: name, Target: "target-" + name, Objective: "objective-" + name}
}

// ── Flow CRUD ──────────────────────────────────────────────────────────

func TestCreateFlow_DefaultTenant(t *testing.T) {
	s := NewMemoryStore()
	flow, err := s.CreateFlow(ctx(), newInput("f1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flow.TenantID != "default" {
		t.Fatalf("expected tenant 'default', got %q", flow.TenantID)
	}
	if flow.Status != domain.StatusPending {
		t.Fatalf("expected status pending, got %s", flow.Status)
	}
	if flow.Name != "f1" || flow.Target != "target-f1" || flow.Objective != "objective-f1" {
		t.Fatalf("flow fields mismatch: %+v", flow)
	}
}

func TestCreateFlowForTenant(t *testing.T) {
	s := NewMemoryStore()
	flow, err := s.CreateFlowForTenant(ctx(), "acme", newInput("f1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flow.TenantID != "acme" {
		t.Fatalf("expected tenant 'acme', got %q", flow.TenantID)
	}
}

func TestGetFlow_HappyPath(t *testing.T) {
	s := NewMemoryStore()
	created, _ := s.CreateFlow(ctx(), newInput("f1"))
	got, err := s.GetFlow(ctx(), created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("expected ID %s, got %s", created.ID, got.ID)
	}
}

func TestGetFlow_NotFound(t *testing.T) {
	s := NewMemoryStore()
	_, err := s.GetFlow(ctx(), "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetFlowForTenant_WrongTenant(t *testing.T) {
	s := NewMemoryStore()
	flow, _ := s.CreateFlowForTenant(ctx(), "acme", newInput("f1"))
	_, err := s.GetFlowForTenant(ctx(), "other", flow.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for wrong tenant, got %v", err)
	}
}

func TestGetFlowForTenant_CorrectTenant(t *testing.T) {
	s := NewMemoryStore()
	flow, _ := s.CreateFlowForTenant(ctx(), "acme", newInput("f1"))
	got, err := s.GetFlowForTenant(ctx(), "acme", flow.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != flow.ID {
		t.Fatalf("expected %s, got %s", flow.ID, got.ID)
	}
}

func TestListFlows(t *testing.T) {
	s := NewMemoryStore()
	s.CreateFlow(ctx(), newInput("a"))
	s.CreateFlow(ctx(), newInput("b"))
	flows, err := s.ListFlows(ctx())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(flows) != 2 {
		t.Fatalf("expected 2 flows, got %d", len(flows))
	}
}

func TestListFlowsByTenant(t *testing.T) {
	s := NewMemoryStore()
	s.CreateFlowForTenant(ctx(), "acme", newInput("a"))
	s.CreateFlowForTenant(ctx(), "acme", newInput("b"))
	s.CreateFlowForTenant(ctx(), "other", newInput("c"))

	flows, err := s.ListFlowsByTenant(ctx(), "acme")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(flows) != 2 {
		t.Fatalf("expected 2 flows for acme, got %d", len(flows))
	}
}

func TestListFlowsPageByTenant(t *testing.T) {
	s := NewMemoryStore()
	for _, name := range []string{"a", "b", "c"} {
		if _, err := s.CreateFlowForTenant(ctx(), "acme", newInput(name)); err != nil {
			t.Fatalf("create flow %s: %v", name, err)
		}
		time.Sleep(2 * time.Millisecond)
	}

	firstPage, nextAfter, hasMore, err := s.ListFlowsPageByTenant(ctx(), "acme", "", 2)
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if len(firstPage) != 2 {
		t.Fatalf("expected 2 flows, got %d", len(firstPage))
	}
	if !hasMore || nextAfter == "" {
		t.Fatalf("expected next cursor, got hasMore=%v nextAfter=%q", hasMore, nextAfter)
	}

	secondPage, nextAfter, hasMore, err := s.ListFlowsPageByTenant(ctx(), "acme", nextAfter, 2)
	if err != nil {
		t.Fatalf("second page: %v", err)
	}
	if len(secondPage) != 1 {
		t.Fatalf("expected 1 flow on second page, got %d", len(secondPage))
	}
	if hasMore || nextAfter != "" {
		t.Fatalf("expected terminal page, got hasMore=%v nextAfter=%q", hasMore, nextAfter)
	}
}

func TestListFlowsPageByTenant_InvalidCursor(t *testing.T) {
	s := NewMemoryStore()
	if _, err := s.CreateFlowForTenant(ctx(), "acme", newInput("a")); err != nil {
		t.Fatalf("create flow: %v", err)
	}

	_, _, _, err := s.ListFlowsPageByTenant(ctx(), "acme", "missing", 2)
	if !errors.Is(err, ErrInvalidPageCursor) {
		t.Fatalf("expected ErrInvalidPageCursor, got %v", err)
	}
}

func TestUpdateFlowStatus(t *testing.T) {
	s := NewMemoryStore()
	flow, _ := s.CreateFlow(ctx(), newInput("f1"))
	updated, err := s.UpdateFlowStatus(ctx(), flow.ID, domain.StatusRunning)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Status != domain.StatusRunning {
		t.Fatalf("expected running, got %s", updated.Status)
	}
	if updated.UpdatedAt.Before(flow.UpdatedAt) {
		t.Fatal("expected UpdatedAt to be at or after original")
	}
}

func TestUpdateFlowStatus_NotFound(t *testing.T) {
	s := NewMemoryStore()
	_, err := s.UpdateFlowStatus(ctx(), "nope", domain.StatusRunning)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// ── Queue & Claim ──────────────────────────────────────────────────────

func TestQueueFlow(t *testing.T) {
	s := NewMemoryStore()
	flow, _ := s.CreateFlow(ctx(), newInput("f1"))
	queued, err := s.QueueFlow(ctx(), flow.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if queued.Status != domain.StatusQueued {
		t.Fatalf("expected queued, got %s", queued.Status)
	}
}

func TestQueueFlowForTenant_WrongTenant(t *testing.T) {
	s := NewMemoryStore()
	flow, _ := s.CreateFlowForTenant(ctx(), "acme", newInput("f1"))
	_, err := s.QueueFlowForTenant(ctx(), "other", flow.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestClaimNextQueuedFlow_ClaimsOldest(t *testing.T) {
	s := NewMemoryStore()

	f1, _ := s.CreateFlow(ctx(), newInput("first"))
	time.Sleep(2 * time.Millisecond)
	f2, _ := s.CreateFlow(ctx(), newInput("second"))

	s.QueueFlow(ctx(), f1.ID)
	s.QueueFlow(ctx(), f2.ID)

	claimed, ok, err := s.ClaimNextQueuedFlow(ctx())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected to claim a flow")
	}
	if claimed.ID != f1.ID {
		t.Fatalf("expected oldest flow %s, got %s", f1.ID, claimed.ID)
	}
	if claimed.Status != domain.StatusRunning {
		t.Fatalf("expected running, got %s", claimed.Status)
	}
}

func TestClaimNextQueuedFlow_NoneQueued(t *testing.T) {
	s := NewMemoryStore()
	s.CreateFlow(ctx(), newInput("f1")) // pending, not queued

	_, ok, err := s.ClaimNextQueuedFlow(ctx())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected no flow to be claimed")
	}
}

func TestClaimNextQueuedFlow_EmptyStore(t *testing.T) {
	s := NewMemoryStore()
	_, ok, err := s.ClaimNextQueuedFlow(ctx())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected no flow in empty store")
	}
}

// ── Agent lifecycle ────────────────────────────────────────────────────

func TestAgentLifecycle(t *testing.T) {
	s := NewMemoryStore()
	flow, _ := s.CreateFlow(ctx(), newInput("f1"))
	agent, err := s.CreateAgent(ctx(), flow.ID, "scanner", "gpt-4")
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if agent.Status != domain.StatusPending {
		t.Fatalf("expected pending, got %s", agent.Status)
	}
	if agent.FlowID != flow.ID || agent.Role != "scanner" || agent.Model != "gpt-4" {
		t.Fatalf("agent fields mismatch: %+v", agent)
	}

	if err := s.UpdateAgentStatus(ctx(), agent.ID, domain.StatusRunning); err != nil {
		t.Fatalf("update agent status: %v", err)
	}

	if err := s.CompleteAgent(ctx(), agent.ID, domain.StatusCompleted); err != nil {
		t.Fatalf("complete agent: %v", err)
	}

	// Verify final status via FlowDetail
	detail, _ := s.FlowDetail(ctx(), flow.ID)
	if len(detail.Agents) != 1 || detail.Agents[0].Status != domain.StatusCompleted {
		t.Fatalf("expected completed agent in detail, got %+v", detail.Agents)
	}
}

func TestUpdateAgentStatus_NotFound(t *testing.T) {
	s := NewMemoryStore()
	err := s.UpdateAgentStatus(ctx(), "nonexistent", domain.StatusRunning)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// ── Task lifecycle ─────────────────────────────────────────────────────

func TestTaskLifecycle(t *testing.T) {
	s := NewMemoryStore()
	flow, _ := s.CreateFlow(ctx(), newInput("f1"))

	task, err := s.CreateTask(ctx(), flow.ID, "recon", "run reconnaissance", "scanner")
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if task.Status != domain.StatusPending {
		t.Fatalf("expected pending, got %s", task.Status)
	}
	if task.FlowID != flow.ID || task.Name != "recon" {
		t.Fatalf("task fields mismatch: %+v", task)
	}

	if err := s.UpdateTaskStatus(ctx(), task.ID, domain.StatusRunning); err != nil {
		t.Fatalf("update task: %v", err)
	}
	if err := s.CompleteTask(ctx(), task.ID, domain.StatusCompleted); err != nil {
		t.Fatalf("complete task: %v", err)
	}
}

func TestUpdateTaskStatus_NotFound(t *testing.T) {
	s := NewMemoryStore()
	err := s.UpdateTaskStatus(ctx(), "nonexistent", domain.StatusRunning)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// ── Subtask lifecycle ──────────────────────────────────────────────────

func TestSubtaskLifecycle(t *testing.T) {
	s := NewMemoryStore()
	flow, _ := s.CreateFlow(ctx(), newInput("f1"))
	task, _ := s.CreateTask(ctx(), flow.ID, "t1", "desc", "role")

	subtask, err := s.CreateSubtask(ctx(), flow.ID, task.ID, "sub1", "subdesc", "subrole")
	if err != nil {
		t.Fatalf("create subtask: %v", err)
	}
	if subtask.Status != domain.StatusPending {
		t.Fatalf("expected pending, got %s", subtask.Status)
	}
	if subtask.FlowID != flow.ID || subtask.TaskID != task.ID {
		t.Fatalf("subtask fields mismatch: %+v", subtask)
	}

	if err := s.UpdateSubtaskStatus(ctx(), subtask.ID, domain.StatusRunning); err != nil {
		t.Fatalf("update subtask: %v", err)
	}
	if err := s.CompleteSubtask(ctx(), subtask.ID, domain.StatusCompleted); err != nil {
		t.Fatalf("complete subtask: %v", err)
	}
}

func TestUpdateSubtaskStatus_NotFound(t *testing.T) {
	s := NewMemoryStore()
	err := s.UpdateSubtaskStatus(ctx(), "nonexistent", domain.StatusRunning)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// ── Action ─────────────────────────────────────────────────────────────

func TestActionCreateAndComplete(t *testing.T) {
	s := NewMemoryStore()
	flow, _ := s.CreateFlow(ctx(), newInput("f1"))
	input := map[string]string{"url": "http://example.com"}

	action, err := s.CreateAction(ctx(), flow.ID, "task1", "sub1", "scanner", "nmap_scan", "docker", input)
	if err != nil {
		t.Fatalf("create action: %v", err)
	}
	if action.Status != domain.StatusRunning {
		t.Fatalf("expected running, got %s", action.Status)
	}
	if action.FunctionName != "nmap_scan" {
		t.Fatalf("expected nmap_scan, got %s", action.FunctionName)
	}
	if action.Input["url"] != "http://example.com" {
		t.Fatalf("input mismatch: %v", action.Input)
	}

	// Mutate original map to verify cloning
	input["url"] = "mutated"
	got, _ := s.FlowDetail(ctx(), flow.ID)
	if got.Actions[0].Input["url"] != "http://example.com" {
		t.Fatal("input was not cloned — mutation leaked")
	}

	output := map[string]string{"result": "open ports: 80,443"}
	completed, err := s.CompleteAction(ctx(), action.ID, domain.StatusCompleted, output)
	if err != nil {
		t.Fatalf("complete action: %v", err)
	}
	if completed.Status != domain.StatusCompleted {
		t.Fatalf("expected completed, got %s", completed.Status)
	}
	if completed.Output["result"] != "open ports: 80,443" {
		t.Fatalf("output mismatch: %v", completed.Output)
	}
}

func TestCompleteAction_NotFound(t *testing.T) {
	s := NewMemoryStore()
	_, err := s.CompleteAction(ctx(), "nonexistent", domain.StatusCompleted, nil)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// ── Execution ──────────────────────────────────────────────────────────

func TestExecutionCreateAndComplete(t *testing.T) {
	s := NewMemoryStore()
	meta := map[string]string{"container": "abc123"}

	exec, err := s.CreateExecution(ctx(), "flow1", "action1", "default", "docker", meta)
	if err != nil {
		t.Fatalf("create execution: %v", err)
	}
	if exec.Status != domain.StatusRunning {
		t.Fatalf("expected running, got %s", exec.Status)
	}
	if exec.Profile != "default" || exec.Runtime != "docker" {
		t.Fatalf("fields mismatch: %+v", exec)
	}

	newMeta := map[string]string{"exit_code": "0"}
	err = s.CompleteExecution(ctx(), exec.ID, domain.StatusCompleted, "updated-profile", "podman", newMeta)
	if err != nil {
		t.Fatalf("complete execution: %v", err)
	}

	// Read back via internal map to verify metadata update
	s.mu.RLock()
	stored := s.executions[exec.ID]
	s.mu.RUnlock()

	if stored.Profile != "updated-profile" {
		t.Fatalf("expected profile 'updated-profile', got %q", stored.Profile)
	}
	if stored.Runtime != "podman" {
		t.Fatalf("expected runtime 'podman', got %q", stored.Runtime)
	}
	if stored.Metadata["exit_code"] != "0" {
		t.Fatalf("metadata not updated: %v", stored.Metadata)
	}
	if stored.CompletedAt.IsZero() {
		t.Fatal("expected CompletedAt to be set")
	}
}

func TestCompleteExecution_PreservesFieldsWhenEmpty(t *testing.T) {
	s := NewMemoryStore()
	exec, _ := s.CreateExecution(ctx(), "flow1", "action1", "original", "docker", nil)

	// Empty strings should NOT overwrite existing profile/runtime
	err := s.CompleteExecution(ctx(), exec.ID, domain.StatusCompleted, "", "", nil)
	if err != nil {
		t.Fatalf("complete execution: %v", err)
	}

	s.mu.RLock()
	stored := s.executions[exec.ID]
	s.mu.RUnlock()

	if stored.Profile != "original" {
		t.Fatalf("expected profile preserved as 'original', got %q", stored.Profile)
	}
	if stored.Runtime != "docker" {
		t.Fatalf("expected runtime preserved as 'docker', got %q", stored.Runtime)
	}
}

func TestCompleteExecution_NotFound(t *testing.T) {
	s := NewMemoryStore()
	err := s.CompleteExecution(ctx(), "nonexistent", domain.StatusCompleted, "", "", nil)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// ── Artifacts & Findings ───────────────────────────────────────────────

func TestAddArtifact(t *testing.T) {
	s := NewMemoryStore()
	meta := map[string]string{"format": "json"}
	artifact, err := s.AddArtifact(ctx(), "flow1", "action1", "report", "scan.json", `{"ports":[80]}`, meta)
	if err != nil {
		t.Fatalf("add artifact: %v", err)
	}
	if artifact.FlowID != "flow1" || artifact.Kind != "report" || artifact.Name != "scan.json" {
		t.Fatalf("artifact fields mismatch: %+v", artifact)
	}
	if artifact.Metadata["format"] != "json" {
		t.Fatalf("metadata mismatch: %v", artifact.Metadata)
	}
	if artifact.Content != `{"ports":[80]}` {
		t.Fatalf("content mismatch: %s", artifact.Content)
	}
}

func TestAddFinding(t *testing.T) {
	s := NewMemoryStore()
	finding, err := s.AddFinding(ctx(), "flow1", "SQL Injection", "critical", "Found in /login endpoint")
	if err != nil {
		t.Fatalf("add finding: %v", err)
	}
	if finding.FlowID != "flow1" || finding.Title != "SQL Injection" {
		t.Fatalf("finding fields mismatch: %+v", finding)
	}
	if finding.Severity != "critical" || finding.Description != "Found in /login endpoint" {
		t.Fatalf("finding content mismatch: %+v", finding)
	}
}

// ── FlowDetail ─────────────────────────────────────────────────────────

func TestFlowDetail_AggregatesAll(t *testing.T) {
	s := NewMemoryStore()
	flow, _ := s.CreateFlow(ctx(), newInput("f1"))
	fid := flow.ID

	s.CreateAgent(ctx(), fid, "scanner", "gpt-4")  //nolint:errcheck
	s.CreateTask(ctx(), fid, "t1", "desc", "role") //nolint:errcheck
	task, _ := s.CreateTask(ctx(), fid, "t2", "desc2", "role2")
	s.CreateSubtask(ctx(), fid, task.ID, "sub1", "subdesc", "subrole")         //nolint:errcheck
	s.CreateAction(ctx(), fid, task.ID, "", "role", "fn", "docker", nil)       //nolint:errcheck
	s.AddArtifact(ctx(), fid, "action1", "report", "r.txt", "content", nil)    //nolint:errcheck
	s.AddMemory(ctx(), fid, "action1", "observation", "some observation", nil) //nolint:errcheck
	s.AddFinding(ctx(), fid, "vuln", "high", "desc")                           //nolint:errcheck
	s.CreateExecution(ctx(), fid, "action1", "default", "docker", nil)         //nolint:errcheck

	detail, err := s.FlowDetail(ctx(), fid)
	if err != nil {
		t.Fatalf("flow detail: %v", err)
	}
	if detail.Flow.ID != fid {
		t.Fatalf("expected flow ID %s", fid)
	}
	if len(detail.Agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(detail.Agents))
	}
	if len(detail.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(detail.Tasks))
	}
	if len(detail.Subtasks) != 1 {
		t.Fatalf("expected 1 subtask, got %d", len(detail.Subtasks))
	}
	if len(detail.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(detail.Actions))
	}
	if len(detail.Artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(detail.Artifacts))
	}
	if len(detail.Memories) != 1 {
		t.Fatalf("expected 1 memory, got %d", len(detail.Memories))
	}
	if len(detail.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(detail.Findings))
	}
	if len(detail.Executions) != 1 {
		t.Fatalf("expected 1 execution, got %d", len(detail.Executions))
	}
}

func TestFlowDetail_NotFound(t *testing.T) {
	s := NewMemoryStore()
	_, err := s.FlowDetail(ctx(), "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestFlowDetailForTenant_WrongTenant(t *testing.T) {
	s := NewMemoryStore()
	flow, _ := s.CreateFlowForTenant(ctx(), "acme", newInput("f1"))
	_, err := s.FlowDetailForTenant(ctx(), "other", flow.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for wrong tenant, got %v", err)
	}
}

func TestFlowDetailForTenant_CorrectTenant(t *testing.T) {
	s := NewMemoryStore()
	flow, _ := s.CreateFlowForTenant(ctx(), "acme", newInput("f1"))
	detail, err := s.FlowDetailForTenant(ctx(), "acme", flow.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.Flow.ID != flow.ID {
		t.Fatalf("expected flow %s, got %s", flow.ID, detail.Flow.ID)
	}
}

func TestFlowDetail_DoesNotLeakOtherFlows(t *testing.T) {
	s := NewMemoryStore()
	f1, _ := s.CreateFlow(ctx(), newInput("f1"))
	f2, _ := s.CreateFlow(ctx(), newInput("f2"))

	s.CreateTask(ctx(), f1.ID, "task-f1", "desc", "role") //nolint:errcheck
	s.CreateTask(ctx(), f2.ID, "task-f2", "desc", "role") //nolint:errcheck

	detail, _ := s.FlowDetail(ctx(), f1.ID)
	if len(detail.Tasks) != 1 {
		t.Fatalf("expected 1 task for f1, got %d", len(detail.Tasks))
	}
	if detail.Tasks[0].Name != "task-f1" {
		t.Fatalf("expected task-f1, got %s", detail.Tasks[0].Name)
	}
}

// ── Approvals ──────────────────────────────────────────────────────────

func TestApprovalLifecycle(t *testing.T) {
	s := NewMemoryStore()
	flow, _ := s.CreateFlowForTenant(ctx(), "acme", newInput("f1"))

	payload := map[string]string{"action": "exploit"}
	approval, err := s.CreateApproval(ctx(), flow.ID, "", "risky_action", "operator", "needs review", payload)
	if err != nil {
		t.Fatalf("create approval: %v", err)
	}
	if approval.Status != "pending" {
		t.Fatalf("expected pending, got %s", approval.Status)
	}
	if approval.TenantID != "acme" {
		t.Fatalf("expected tenant from flow 'acme', got %q", approval.TenantID)
	}
	if approval.RequestedBy != "operator" {
		t.Fatalf("expected requestedBy 'operator', got %q", approval.RequestedBy)
	}

	// Get
	got, err := s.GetApproval(ctx(), approval.ID)
	if err != nil {
		t.Fatalf("get approval: %v", err)
	}
	if got.ID != approval.ID {
		t.Fatalf("expected %s, got %s", approval.ID, got.ID)
	}

	// List by tenant
	byTenant, err := s.ListApprovalsByTenant(ctx(), "acme")
	if err != nil {
		t.Fatalf("list by tenant: %v", err)
	}
	if len(byTenant) != 1 {
		t.Fatalf("expected 1 approval for acme, got %d", len(byTenant))
	}

	// List by flow
	byFlow, err := s.ListApprovalsByFlow(ctx(), flow.ID)
	if err != nil {
		t.Fatalf("list by flow: %v", err)
	}
	if len(byFlow) != 1 {
		t.Fatalf("expected 1 approval for flow, got %d", len(byFlow))
	}
}

func TestApprovalCreateWithExplicitTenant(t *testing.T) {
	s := NewMemoryStore()
	flow, _ := s.CreateFlowForTenant(ctx(), "acme", newInput("f1"))

	approval, err := s.CreateApproval(ctx(), flow.ID, "override-tenant", "risky", "op", "reason", nil)
	if err != nil {
		t.Fatalf("create approval: %v", err)
	}
	if approval.TenantID != "override-tenant" {
		t.Fatalf("expected override-tenant, got %q", approval.TenantID)
	}
}

func TestListApprovalsPageByTenant(t *testing.T) {
	s := NewMemoryStore()
	flow, _ := s.CreateFlowForTenant(ctx(), "acme", newInput("f1"))
	for i := 0; i < 3; i++ {
		if _, err := s.CreateApproval(ctx(), flow.ID, "", "risky", "operator", "needs review", nil); err != nil {
			t.Fatalf("create approval %d: %v", i, err)
		}
		time.Sleep(2 * time.Millisecond)
	}

	firstPage, nextAfter, hasMore, err := s.ListApprovalsPageByTenant(ctx(), "acme", "", 2)
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if len(firstPage) != 2 {
		t.Fatalf("expected 2 approvals, got %d", len(firstPage))
	}
	if !hasMore || nextAfter == "" {
		t.Fatalf("expected next cursor, got hasMore=%v nextAfter=%q", hasMore, nextAfter)
	}

	secondPage, nextAfter, hasMore, err := s.ListApprovalsPageByTenant(ctx(), "acme", nextAfter, 2)
	if err != nil {
		t.Fatalf("second page: %v", err)
	}
	if len(secondPage) != 1 {
		t.Fatalf("expected 1 approval on second page, got %d", len(secondPage))
	}
	if hasMore || nextAfter != "" {
		t.Fatalf("expected terminal page, got hasMore=%v nextAfter=%q", hasMore, nextAfter)
	}
}

func TestApprovalAnonymousRequestedBy(t *testing.T) {
	s := NewMemoryStore()
	flow, _ := s.CreateFlow(ctx(), newInput("f1"))

	approval, err := s.CreateApproval(ctx(), flow.ID, "", "risky", "", "reason", nil)
	if err != nil {
		t.Fatalf("create approval: %v", err)
	}
	if approval.RequestedBy != "anonymous" {
		t.Fatalf("expected 'anonymous', got %q", approval.RequestedBy)
	}
}

func TestGetApproval_NotFound(t *testing.T) {
	s := NewMemoryStore()
	_, err := s.GetApproval(ctx(), "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestReviewApproval_Approve(t *testing.T) {
	s := NewMemoryStore()
	flow, _ := s.CreateFlow(ctx(), newInput("f1"))
	approval, _ := s.CreateApproval(ctx(), flow.ID, "", "risky", "op", "reason", nil)

	reviewed, err := s.ReviewApproval(ctx(), approval.ID, true, "admin", "looks good")
	if err != nil {
		t.Fatalf("review approval: %v", err)
	}
	if reviewed.Status != "approved" {
		t.Fatalf("expected approved, got %s", reviewed.Status)
	}
	if reviewed.ReviewedBy != "admin" {
		t.Fatalf("expected admin, got %q", reviewed.ReviewedBy)
	}
	if reviewed.ReviewNote != "looks good" {
		t.Fatalf("expected 'looks good', got %q", reviewed.ReviewNote)
	}
	if reviewed.ReviewedAt.IsZero() {
		t.Fatal("expected ReviewedAt to be set")
	}
}

func TestReviewApproval_Reject(t *testing.T) {
	s := NewMemoryStore()
	flow, _ := s.CreateFlow(ctx(), newInput("f1"))
	approval, _ := s.CreateApproval(ctx(), flow.ID, "", "risky", "op", "reason", nil)

	reviewed, err := s.ReviewApproval(ctx(), approval.ID, false, "admin", "too risky")
	if err != nil {
		t.Fatalf("review approval: %v", err)
	}
	if reviewed.Status != "rejected" {
		t.Fatalf("expected rejected, got %s", reviewed.Status)
	}
}

func TestReviewApproval_AlreadyReviewed(t *testing.T) {
	s := NewMemoryStore()
	flow, _ := s.CreateFlow(ctx(), newInput("f1"))
	approval, _ := s.CreateApproval(ctx(), flow.ID, "", "risky", "op", "reason", nil)

	s.ReviewApproval(ctx(), approval.ID, true, "admin", "ok") //nolint:errcheck
	// Review again — should be a no-op
	second, err := s.ReviewApproval(ctx(), approval.ID, false, "other", "nope")
	if err != nil {
		t.Fatalf("review approval: %v", err)
	}
	if second.Status != "approved" {
		t.Fatalf("expected status unchanged (approved), got %s", second.Status)
	}
}

func TestReviewApproval_NotFound(t *testing.T) {
	s := NewMemoryStore()
	_, err := s.ReviewApproval(ctx(), "nonexistent", true, "admin", "")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestReviewApproval_AnonymousReviewer(t *testing.T) {
	s := NewMemoryStore()
	flow, _ := s.CreateFlow(ctx(), newInput("f1"))
	approval, _ := s.CreateApproval(ctx(), flow.ID, "", "risky", "op", "reason", nil)

	reviewed, _ := s.ReviewApproval(ctx(), approval.ID, true, "", "")
	if reviewed.ReviewedBy != "anonymous" {
		t.Fatalf("expected anonymous reviewer, got %q", reviewed.ReviewedBy)
	}
}

// ── Events ─────────────────────────────────────────────────────────────

func TestRecordEvent_AutoIncrement(t *testing.T) {
	s := NewMemoryStore()
	e1, err := s.RecordEvent(ctx(), "flow1", "action.started", "msg1", nil)
	if err != nil {
		t.Fatalf("record event: %v", err)
	}
	e2, err := s.RecordEvent(ctx(), "flow1", "action.completed", "msg2", nil)
	if err != nil {
		t.Fatalf("record event: %v", err)
	}
	if e2.Sequence <= e1.Sequence {
		t.Fatalf("expected auto-incrementing sequence: %d <= %d", e2.Sequence, e1.Sequence)
	}
}

func TestListEvents_FiltersByFlowID(t *testing.T) {
	s := NewMemoryStore()
	s.RecordEvent(ctx(), "flow1", "a", "msg", nil) //nolint:errcheck
	s.RecordEvent(ctx(), "flow2", "b", "msg", nil) //nolint:errcheck
	s.RecordEvent(ctx(), "flow1", "c", "msg", nil) //nolint:errcheck

	events, err := s.ListEvents(ctx(), "flow1", 0)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events for flow1, got %d", len(events))
	}
	for _, e := range events {
		if e.FlowID != "flow1" {
			t.Fatalf("expected flow1, got %s", e.FlowID)
		}
	}
}

func TestListEvents_FiltersBySequence(t *testing.T) {
	s := NewMemoryStore()
	e1, _ := s.RecordEvent(ctx(), "flow1", "a", "msg1", nil)
	s.RecordEvent(ctx(), "flow1", "b", "msg2", nil)
	s.RecordEvent(ctx(), "flow1", "c", "msg3", nil)

	events, err := s.ListEvents(ctx(), "flow1", e1.Sequence)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events after seq %d, got %d", e1.Sequence, len(events))
	}
	for _, e := range events {
		if e.Sequence <= e1.Sequence {
			t.Fatalf("event sequence %d should be > %d", e.Sequence, e1.Sequence)
		}
	}
}

func TestListEvents_Empty(t *testing.T) {
	s := NewMemoryStore()
	events, err := s.ListEvents(ctx(), "flow1", 0)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}
}

func TestRecordEvent_PayloadCloned(t *testing.T) {
	s := NewMemoryStore()
	payload := map[string]any{"key": "value"}
	e, _ := s.RecordEvent(ctx(), "flow1", "a", "msg", payload)

	payload["key"] = "mutated"

	events, _ := s.ListEvents(ctx(), "flow1", 0)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	// Note: CreateFlow also records an event, but that's on a different flowID.
	// RecordEvent with flowID "flow1" should only have our event.
	if events[0].Payload["key"] != "value" {
		t.Fatalf("payload was not cloned — mutation leaked: %v (event ID %s)", events[0].Payload, e.ID)
	}
}

// ── CreateFlow records event ───────────────────────────────────────────

func TestCreateFlow_RecordsEvent(t *testing.T) {
	s := NewMemoryStore()
	flow, _ := s.CreateFlow(ctx(), newInput("f1"))

	events, err := s.ListEvents(ctx(), flow.ID, 0)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 flow.created event, got %d", len(events))
	}
	if events[0].Type != "flow.created" {
		t.Fatalf("expected event type 'flow.created', got %q", events[0].Type)
	}
}

// ── Tenant normalization edge cases ────────────────────────────────────

func TestCreateFlowForTenant_EmptyTenantNormalizesToDefault(t *testing.T) {
	s := NewMemoryStore()
	flow, _ := s.CreateFlowForTenant(ctx(), "", newInput("f1"))
	if flow.TenantID != "default" {
		t.Fatalf("expected 'default' for empty tenant, got %q", flow.TenantID)
	}
}

func TestCreateFlowForTenant_WhitespaceTenantNormalizesToDefault(t *testing.T) {
	s := NewMemoryStore()
	flow, _ := s.CreateFlowForTenant(ctx(), "   ", newInput("f1"))
	if flow.TenantID != "default" {
		t.Fatalf("expected 'default' for whitespace tenant, got %q", flow.TenantID)
	}
}
