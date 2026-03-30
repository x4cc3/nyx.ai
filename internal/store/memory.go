package store

import (
	"context"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"nyx/internal/domain"
	"nyx/internal/ids"
	"nyx/internal/memvec"
)

type MemoryStore struct {
	mu         sync.RWMutex
	flows      map[string]domain.Flow
	tasks      map[string]domain.Task
	subtasks   map[string]domain.Subtask
	actions    map[string]domain.Action
	artifacts  map[string]domain.Artifact
	memories   map[string]domain.Memory
	findings   map[string]domain.Finding
	agents     map[string]domain.Agent
	executions map[string]domain.Execution
	embeddings map[string][]float32
	approvals  map[string]domain.Approval
	events     []domain.Event
	nextSeq    int64
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		flows:      make(map[string]domain.Flow),
		tasks:      make(map[string]domain.Task),
		subtasks:   make(map[string]domain.Subtask),
		actions:    make(map[string]domain.Action),
		artifacts:  make(map[string]domain.Artifact),
		memories:   make(map[string]domain.Memory),
		findings:   make(map[string]domain.Finding),
		agents:     make(map[string]domain.Agent),
		executions: make(map[string]domain.Execution),
		embeddings: make(map[string][]float32),
		approvals:  make(map[string]domain.Approval),
		events:     make([]domain.Event, 0),
	}
}

func (s *MemoryStore) Init(context.Context) error { return nil }
func (s *MemoryStore) Ping(context.Context) error { return nil }
func (s *MemoryStore) Close() error               { return nil }

func (s *MemoryStore) CreateFlow(ctx context.Context, input domain.CreateFlowInput) (domain.Flow, error) {
	return s.CreateFlowForTenant(ctx, "default", input)
}

func (s *MemoryStore) CreateFlowForTenant(ctx context.Context, tenantID string, input domain.CreateFlowInput) (domain.Flow, error) {
	now := time.Now().UTC()
	flow := domain.Flow{
		ID:        ids.New("flow"),
		TenantID:  normalizeTenant(tenantID),
		Name:      input.Name,
		Target:    input.Target,
		Objective: input.Objective,
		Status:    domain.StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.mu.Lock()
	s.flows[flow.ID] = flow
	s.mu.Unlock()

	_, _ = s.RecordEvent(ctx, flow.ID, domain.EventFlowCreated, "Flow created", map[string]any{"status": flow.Status})
	return flow, nil
}

func (s *MemoryStore) ListFlows(ctx context.Context) ([]domain.Flow, error) {
	return s.listFlows("")
}

func (s *MemoryStore) ListFlowsByTenant(ctx context.Context, tenantID string) ([]domain.Flow, error) {
	return s.listFlows(normalizeTenant(tenantID))
}

func (s *MemoryStore) ListFlowsPageByTenant(ctx context.Context, tenantID, afterID string, limit int) ([]domain.Flow, string, bool, error) {
	flows, err := s.ListFlowsByTenant(ctx, tenantID)
	if err != nil {
		return nil, "", false, err
	}
	return pageByID(flows, afterID, limit, func(flow domain.Flow) string { return flow.ID })
}

func (s *MemoryStore) listFlows(tenantID string) ([]domain.Flow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]domain.Flow, 0, len(s.flows))
	for _, flow := range s.flows {
		if tenantID != "" && flow.TenantID != tenantID {
			continue
		}
		out = append(out, flow)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID > out[j].ID
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func (s *MemoryStore) GetFlow(ctx context.Context, id string) (domain.Flow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	flow, ok := s.flows[id]
	if !ok {
		return domain.Flow{}, ErrNotFound
	}
	return flow, nil
}

func (s *MemoryStore) GetFlowForTenant(ctx context.Context, tenantID, id string) (domain.Flow, error) {
	flow, err := s.GetFlow(ctx, id)
	if err != nil {
		return domain.Flow{}, err
	}
	if flow.TenantID != normalizeTenant(tenantID) {
		return domain.Flow{}, ErrNotFound
	}
	return flow, nil
}

func (s *MemoryStore) QueueFlow(ctx context.Context, id string) (domain.Flow, error) {
	return s.UpdateFlowStatus(ctx, id, domain.StatusQueued)
}

func (s *MemoryStore) QueueFlowForTenant(ctx context.Context, tenantID, id string) (domain.Flow, error) {
	if _, err := s.GetFlowForTenant(ctx, tenantID, id); err != nil {
		return domain.Flow{}, err
	}
	return s.QueueFlow(ctx, id)
}

func (s *MemoryStore) UpdateFlowStatus(ctx context.Context, id string, status domain.Status) (domain.Flow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	flow, ok := s.flows[id]
	if !ok {
		return domain.Flow{}, ErrNotFound
	}
	flow.Status = status
	flow.UpdatedAt = time.Now().UTC()
	s.flows[id] = flow
	return flow, nil
}

func (s *MemoryStore) ClaimNextQueuedFlow(ctx context.Context) (domain.Flow, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var chosen domain.Flow
	found := false
	for _, flow := range s.flows {
		if flow.Status != domain.StatusQueued {
			continue
		}
		if !found || flow.CreatedAt.Before(chosen.CreatedAt) {
			chosen = flow
			found = true
		}
	}
	if !found {
		return domain.Flow{}, false, nil
	}
	chosen.Status = domain.StatusRunning
	chosen.UpdatedAt = time.Now().UTC()
	s.flows[chosen.ID] = chosen
	return chosen, true, nil
}

func (s *MemoryStore) CreateAgent(ctx context.Context, flowID, role, model string) (domain.Agent, error) {
	now := time.Now().UTC()
	agent := domain.Agent{
		ID:        ids.New("agent"),
		FlowID:    flowID,
		Role:      role,
		Model:     model,
		Status:    domain.StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.agents[agent.ID] = agent
	return agent, nil
}

func (s *MemoryStore) UpdateAgentStatus(ctx context.Context, id string, status domain.Status) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	agent, ok := s.agents[id]
	if !ok {
		return ErrNotFound
	}
	agent.Status = status
	agent.UpdatedAt = time.Now().UTC()
	s.agents[id] = agent
	return nil
}

func (s *MemoryStore) CompleteAgent(ctx context.Context, id string, status domain.Status) error {
	return s.UpdateAgentStatus(ctx, id, status)
}

func (s *MemoryStore) CreateTask(ctx context.Context, flowID, name, description, role string) (domain.Task, error) {
	now := time.Now().UTC()
	task := domain.Task{
		ID:          ids.New("task"),
		FlowID:      flowID,
		Name:        name,
		Description: description,
		AgentRole:   role,
		Status:      domain.StatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[task.ID] = task
	return task, nil
}

func (s *MemoryStore) UpdateTaskStatus(ctx context.Context, id string, status domain.Status) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	task, ok := s.tasks[id]
	if !ok {
		return ErrNotFound
	}
	task.Status = status
	task.UpdatedAt = time.Now().UTC()
	s.tasks[id] = task
	return nil
}

func (s *MemoryStore) CompleteTask(ctx context.Context, id string, status domain.Status) error {
	return s.UpdateTaskStatus(ctx, id, status)
}

func (s *MemoryStore) CreateSubtask(ctx context.Context, flowID, taskID, name, description, role string) (domain.Subtask, error) {
	now := time.Now().UTC()
	subtask := domain.Subtask{
		ID:          ids.New("subtask"),
		FlowID:      flowID,
		TaskID:      taskID,
		Name:        name,
		Description: description,
		AgentRole:   role,
		Status:      domain.StatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subtasks[subtask.ID] = subtask
	return subtask, nil
}

func (s *MemoryStore) UpdateSubtaskStatus(ctx context.Context, id string, status domain.Status) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	subtask, ok := s.subtasks[id]
	if !ok {
		return ErrNotFound
	}
	subtask.Status = status
	subtask.UpdatedAt = time.Now().UTC()
	s.subtasks[id] = subtask
	return nil
}

func (s *MemoryStore) CompleteSubtask(ctx context.Context, id string, status domain.Status) error {
	return s.UpdateSubtaskStatus(ctx, id, status)
}

func (s *MemoryStore) CreateAction(ctx context.Context, flowID, taskID, subtaskID, role, functionName, executionMode string, input map[string]string) (domain.Action, error) {
	now := time.Now().UTC()
	action := domain.Action{
		ID:            ids.New("action"),
		FlowID:        flowID,
		TaskID:        taskID,
		SubtaskID:     subtaskID,
		AgentRole:     role,
		FunctionName:  functionName,
		Input:         cloneStringMap(input),
		Status:        domain.StatusRunning,
		ExecutionMode: executionMode,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.actions[action.ID] = action
	return action, nil
}

func (s *MemoryStore) CompleteAction(ctx context.Context, id string, status domain.Status, output map[string]string) (domain.Action, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	action, ok := s.actions[id]
	if !ok {
		return domain.Action{}, ErrNotFound
	}
	action.Status = status
	action.Output = cloneStringMap(output)
	action.UpdatedAt = time.Now().UTC()
	s.actions[id] = action
	return action, nil
}

func (s *MemoryStore) CreateExecution(ctx context.Context, flowID, actionID, profile, runtime string, metadata map[string]string) (domain.Execution, error) {
	now := time.Now().UTC()
	exec := domain.Execution{
		ID:        ids.New("exec"),
		FlowID:    flowID,
		ActionID:  actionID,
		Profile:   profile,
		Runtime:   runtime,
		Metadata:  cloneStringMap(metadata),
		Status:    domain.StatusRunning,
		StartedAt: now,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.executions[exec.ID] = exec
	return exec, nil
}

func (s *MemoryStore) CompleteExecution(ctx context.Context, id string, status domain.Status, profile, runtime string, metadata map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	exec, ok := s.executions[id]
	if !ok {
		return ErrNotFound
	}
	if trimmed := strings.TrimSpace(profile); trimmed != "" {
		exec.Profile = trimmed
	}
	if trimmed := strings.TrimSpace(runtime); trimmed != "" {
		exec.Runtime = trimmed
	}
	if metadata != nil {
		exec.Metadata = cloneStringMap(metadata)
	}
	exec.Status = status
	exec.CompletedAt = time.Now().UTC()
	s.executions[id] = exec
	return nil
}

func (s *MemoryStore) AddArtifact(ctx context.Context, flowID, actionID, kind, name, content string, metadata map[string]string) (domain.Artifact, error) {
	artifact := domain.Artifact{
		ID:        ids.New("artifact"),
		FlowID:    flowID,
		ActionID:  actionID,
		Kind:      kind,
		Name:      name,
		Content:   content,
		Metadata:  cloneStringMap(metadata),
		CreatedAt: time.Now().UTC(),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.artifacts[artifact.ID] = artifact
	return artifact, nil
}

func (s *MemoryStore) AddMemory(ctx context.Context, flowID, actionID, kind, content string, metadata map[string]string) (domain.Memory, error) {
	content, metadata, embedding := memvec.Prepare(kind, content, metadata)
	memory := domain.Memory{
		ID:        ids.New("memory"),
		FlowID:    flowID,
		ActionID:  actionID,
		Kind:      kind,
		Content:   content,
		Metadata:  cloneStringMap(metadata),
		CreatedAt: time.Now().UTC(),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.memories[memory.ID] = memory
	s.embeddings[memory.ID] = embedding
	return memory, nil
}

func (s *MemoryStore) AddFinding(ctx context.Context, flowID, title, severity, description string) (domain.Finding, error) {
	finding := domain.Finding{
		ID:          ids.New("finding"),
		FlowID:      flowID,
		Title:       title,
		Severity:    severity,
		Description: description,
		CreatedAt:   time.Now().UTC(),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.findings[finding.ID] = finding
	return finding, nil
}

func (s *MemoryStore) FlowDetail(ctx context.Context, id string) (domain.FlowDetail, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	flow, ok := s.flows[id]
	if !ok {
		return domain.FlowDetail{}, ErrNotFound
	}

	detail := domain.FlowDetail{Flow: flow}
	for _, task := range s.tasks {
		if task.FlowID == id {
			detail.Tasks = append(detail.Tasks, task)
		}
	}
	for _, subtask := range s.subtasks {
		if subtask.FlowID == id {
			detail.Subtasks = append(detail.Subtasks, subtask)
		}
	}
	for _, action := range s.actions {
		if action.FlowID == id {
			detail.Actions = append(detail.Actions, action)
		}
	}
	for _, artifact := range s.artifacts {
		if artifact.FlowID == id {
			detail.Artifacts = append(detail.Artifacts, artifact)
		}
	}
	for _, memory := range s.memories {
		if memory.FlowID == id {
			detail.Memories = append(detail.Memories, memory)
		}
	}
	for _, finding := range s.findings {
		if finding.FlowID == id {
			detail.Findings = append(detail.Findings, finding)
		}
	}
	for _, agent := range s.agents {
		if agent.FlowID == id {
			detail.Agents = append(detail.Agents, agent)
		}
	}
	for _, execution := range s.executions {
		if execution.FlowID == id {
			detail.Executions = append(detail.Executions, execution)
		}
	}

	sort.Slice(detail.Tasks, func(i, j int) bool { return detail.Tasks[i].CreatedAt.Before(detail.Tasks[j].CreatedAt) })
	sort.Slice(detail.Subtasks, func(i, j int) bool { return detail.Subtasks[i].CreatedAt.Before(detail.Subtasks[j].CreatedAt) })
	sort.Slice(detail.Actions, func(i, j int) bool { return detail.Actions[i].CreatedAt.Before(detail.Actions[j].CreatedAt) })
	sort.Slice(detail.Artifacts, func(i, j int) bool { return detail.Artifacts[i].CreatedAt.Before(detail.Artifacts[j].CreatedAt) })
	sort.Slice(detail.Memories, func(i, j int) bool { return detail.Memories[i].CreatedAt.Before(detail.Memories[j].CreatedAt) })
	sort.Slice(detail.Findings, func(i, j int) bool { return detail.Findings[i].CreatedAt.Before(detail.Findings[j].CreatedAt) })
	sort.Slice(detail.Agents, func(i, j int) bool { return detail.Agents[i].CreatedAt.Before(detail.Agents[j].CreatedAt) })
	sort.Slice(detail.Executions, func(i, j int) bool { return detail.Executions[i].StartedAt.Before(detail.Executions[j].StartedAt) })

	return detail, nil
}

func (s *MemoryStore) FlowDetailForTenant(ctx context.Context, tenantID, id string) (domain.FlowDetail, error) {
	detail, err := s.FlowDetail(ctx, id)
	if err != nil {
		return domain.FlowDetail{}, err
	}
	if detail.Flow.TenantID != normalizeTenant(tenantID) {
		return domain.FlowDetail{}, ErrNotFound
	}
	return detail, nil
}

func (s *MemoryStore) SearchMemories(ctx context.Context, flowID, query string) ([]domain.Memory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query = strings.ToLower(strings.TrimSpace(query))
	type scoredMemory struct {
		memory domain.Memory
		score  float64
	}
	results := make([]scoredMemory, 0)
	queryEmbedding := memvec.Embed(query)
	for _, memory := range s.memories {
		if memory.FlowID != flowID {
			continue
		}
		score := semanticScore(query, queryEmbedding, memory, s.embeddings[memory.ID])
		if query == "" || score > 0 {
			results = append(results, scoredMemory{memory: memory, score: score})
		}
	}
	sort.Slice(results, func(i, j int) bool {
		if math.Abs(results[i].score-results[j].score) < 0.0001 {
			return results[i].memory.CreatedAt.After(results[j].memory.CreatedAt)
		}
		return results[i].score > results[j].score
	})
	if len(results) > 8 {
		results = results[:8]
	}
	out := make([]domain.Memory, 0, len(results))
	for _, item := range results {
		out = append(out, item.memory)
	}
	return out, nil
}

func (s *MemoryStore) CreateApproval(ctx context.Context, flowID, tenantID, kind, requestedBy, reason string, payload map[string]string) (domain.Approval, error) {
	flow, err := s.GetFlow(ctx, flowID)
	if err != nil {
		return domain.Approval{}, err
	}
	approval := domain.Approval{
		ID:          ids.New("approval"),
		FlowID:      flowID,
		TenantID:    flow.TenantID,
		Kind:        kind,
		Status:      "pending",
		RequestedBy: strings.TrimSpace(requestedBy),
		Reason:      reason,
		Payload:     cloneStringMap(payload),
		CreatedAt:   time.Now().UTC(),
	}
	if strings.TrimSpace(tenantID) != "" {
		approval.TenantID = normalizeTenant(tenantID)
	}
	if approval.RequestedBy == "" {
		approval.RequestedBy = "anonymous"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.approvals[approval.ID] = approval
	return approval, nil
}

func (s *MemoryStore) GetApproval(ctx context.Context, id string) (domain.Approval, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	approval, ok := s.approvals[id]
	if !ok {
		return domain.Approval{}, ErrNotFound
	}
	return approval, nil
}

func (s *MemoryStore) ListApprovalsByTenant(ctx context.Context, tenantID string) ([]domain.Approval, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tenantID = normalizeTenant(tenantID)
	out := make([]domain.Approval, 0)
	for _, approval := range s.approvals {
		if approval.TenantID != tenantID {
			continue
		}
		out = append(out, approval)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID > out[j].ID
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func (s *MemoryStore) ListApprovalsPageByTenant(ctx context.Context, tenantID, afterID string, limit int) ([]domain.Approval, string, bool, error) {
	approvals, err := s.ListApprovalsByTenant(ctx, tenantID)
	if err != nil {
		return nil, "", false, err
	}
	return pageByID(approvals, afterID, limit, func(approval domain.Approval) string { return approval.ID })
}

func (s *MemoryStore) ListApprovalsByFlow(ctx context.Context, flowID string) ([]domain.Approval, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.Approval, 0)
	for _, approval := range s.approvals {
		if approval.FlowID != flowID {
			continue
		}
		out = append(out, approval)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (s *MemoryStore) ReviewApproval(ctx context.Context, id string, approved bool, reviewedBy, note string) (domain.Approval, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	approval, ok := s.approvals[id]
	if !ok {
		return domain.Approval{}, ErrNotFound
	}
	if approval.Status != "pending" {
		return approval, nil
	}
	if approved {
		approval.Status = "approved"
	} else {
		approval.Status = "rejected"
	}
	approval.ReviewedBy = strings.TrimSpace(reviewedBy)
	if approval.ReviewedBy == "" {
		approval.ReviewedBy = "anonymous"
	}
	approval.ReviewNote = strings.TrimSpace(note)
	approval.ReviewedAt = time.Now().UTC()
	s.approvals[id] = approval
	return approval, nil
}

func semanticScore(query string, queryEmbedding []float32, memory domain.Memory, memoryEmbedding []float32) float64 {
	if query == "" {
		return 1
	}
	score := memvec.Similarity(queryEmbedding, memoryEmbedding)
	content := strings.ToLower(memory.Content)
	if strings.Contains(content, query) {
		score += 0.5
	}
	if strings.Contains(strings.ToLower(memory.Metadata["function_name"]), query) {
		score += 0.2
	}
	return score
}

func (s *MemoryStore) RecordEvent(ctx context.Context, flowID, eventType, message string, payload map[string]any) (domain.Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextSeq++
	event := domain.Event{
		ID:        ids.New("evt"),
		Sequence:  s.nextSeq,
		FlowID:    flowID,
		Type:      eventType,
		Message:   message,
		Payload:   cloneAnyMap(payload),
		CreatedAt: time.Now().UTC(),
	}
	s.events = append(s.events, event)
	return event, nil
}

func (s *MemoryStore) ListEvents(ctx context.Context, flowID string, afterSequence int64) ([]domain.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	events := make([]domain.Event, 0)
	for _, event := range s.events {
		if event.FlowID == flowID && event.Sequence > afterSequence {
			events = append(events, event)
		}
	}
	return events, nil
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cloneAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func normalizeTenant(tenantID string) string {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return "default"
	}
	return tenantID
}

func pageByID[T any](items []T, afterID string, limit int, id func(T) string) ([]T, string, bool, error) {
	if limit < 1 {
		limit = 1
	}
	start := 0
	afterID = strings.TrimSpace(afterID)
	if afterID != "" {
		start = -1
		for idx, item := range items {
			if id(item) == afterID {
				start = idx + 1
				break
			}
		}
		if start == -1 {
			return nil, "", false, ErrInvalidPageCursor
		}
	}
	if start > len(items) {
		start = len(items)
	}
	end := start + limit
	hasMore := end < len(items)
	if end > len(items) {
		end = len(items)
	}
	pageItems := items[start:end]
	nextAfter := ""
	if hasMore && len(pageItems) > 0 {
		nextAfter = id(pageItems[len(pageItems)-1])
	}
	return pageItems, nextAfter, hasMore, nil
}
