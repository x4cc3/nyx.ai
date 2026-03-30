package store

import (
	"database/sql"
	"encoding/json"
	"time"

	"nyx/internal/dbgen"
	"nyx/internal/domain"
)

func toJSON(value any) []byte {
	if value == nil {
		return []byte("{}")
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return []byte("{}")
	}
	return raw
}

func toStringMap(raw []byte) map[string]string {
	if len(raw) == 0 {
		return map[string]string{}
	}
	var out map[string]string
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]string{}
	}
	return out
}

func toAnyMap(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func toDomainTask(row dbgen.Task) domain.Task {
	return domain.Task{
		ID:          row.ID,
		FlowID:      row.FlowID,
		Name:        row.Name,
		Description: row.Description,
		Status:      domain.Status(row.Status),
		AgentRole:   row.AgentRole,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func toDomainTasks(rows []dbgen.Task) []domain.Task {
	out := make([]domain.Task, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainTask(row))
	}
	return out
}

func toDomainSubtask(row dbgen.Subtask) domain.Subtask {
	return domain.Subtask{
		ID:          row.ID,
		TaskID:      row.TaskID,
		FlowID:      row.FlowID,
		Name:        row.Name,
		Description: row.Description,
		Status:      domain.Status(row.Status),
		AgentRole:   row.AgentRole,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func toDomainSubtasks(rows []dbgen.Subtask) []domain.Subtask {
	out := make([]domain.Subtask, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainSubtask(row))
	}
	return out
}

func toDomainAction(row dbgen.Action) domain.Action {
	return domain.Action{
		ID:            row.ID,
		FlowID:        row.FlowID,
		TaskID:        row.TaskID,
		SubtaskID:     row.SubtaskID,
		AgentRole:     row.AgentRole,
		FunctionName:  row.FunctionName,
		Input:         toStringMap(row.Input),
		Output:        toStringMap(row.Output),
		Status:        domain.Status(row.Status),
		ExecutionMode: row.ExecutionMode,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}

func toDomainActions(rows []dbgen.Action) []domain.Action {
	out := make([]domain.Action, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainAction(row))
	}
	return out
}

func toDomainArtifact(row dbgen.Artifact) domain.Artifact {
	return domain.Artifact{
		ID:        row.ID,
		FlowID:    row.FlowID,
		ActionID:  row.ActionID,
		Kind:      row.Kind,
		Name:      row.Name,
		Content:   row.Content,
		Metadata:  toStringMap(row.Metadata),
		CreatedAt: row.CreatedAt,
	}
}

func toDomainArtifacts(rows []dbgen.Artifact) []domain.Artifact {
	out := make([]domain.Artifact, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainArtifact(row))
	}
	return out
}

type memoryRow interface {
	GetID() string
	GetFlowID() string
	GetActionID() string
	GetKind() string
	GetContent() string
	GetMetadata() []byte
	GetCreatedAt() time.Time
}

type listMemoryRow dbgen.ListMemoriesByFlowRow

func (r listMemoryRow) GetID() string           { return r.ID }
func (r listMemoryRow) GetFlowID() string       { return r.FlowID }
func (r listMemoryRow) GetActionID() string     { return r.ActionID }
func (r listMemoryRow) GetKind() string         { return r.Kind }
func (r listMemoryRow) GetContent() string      { return r.Content }
func (r listMemoryRow) GetMetadata() []byte     { return r.Metadata }
func (r listMemoryRow) GetCreatedAt() time.Time { return r.CreatedAt }

type searchMemoryRow dbgen.SearchMemoriesRow

func (r searchMemoryRow) GetID() string           { return r.ID }
func (r searchMemoryRow) GetFlowID() string       { return r.FlowID }
func (r searchMemoryRow) GetActionID() string     { return r.ActionID }
func (r searchMemoryRow) GetKind() string         { return r.Kind }
func (r searchMemoryRow) GetContent() string      { return r.Content }
func (r searchMemoryRow) GetMetadata() []byte     { return r.Metadata }
func (r searchMemoryRow) GetCreatedAt() time.Time { return r.CreatedAt }

func toDomainMemory(row memoryRow) domain.Memory {
	return domain.Memory{
		ID:        row.GetID(),
		FlowID:    row.GetFlowID(),
		ActionID:  row.GetActionID(),
		Kind:      row.GetKind(),
		Content:   row.GetContent(),
		Metadata:  toStringMap(row.GetMetadata()),
		CreatedAt: row.GetCreatedAt(),
	}
}

func toDomainMemories(rows []dbgen.ListMemoriesByFlowRow) []domain.Memory {
	out := make([]domain.Memory, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainMemory(listMemoryRow(row)))
	}
	return out
}

func toSearchDomainMemories(rows []dbgen.SearchMemoriesRow) []domain.Memory {
	out := make([]domain.Memory, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainMemory(searchMemoryRow(row)))
	}
	return out
}

func toDomainFinding(row dbgen.Finding) domain.Finding {
	return domain.Finding{
		ID:          row.ID,
		FlowID:      row.FlowID,
		Title:       row.Title,
		Severity:    row.Severity,
		Description: row.Description,
		CreatedAt:   row.CreatedAt,
	}
}

func toDomainFindings(rows []dbgen.Finding) []domain.Finding {
	out := make([]domain.Finding, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainFinding(row))
	}
	return out
}

func toDomainAgent(row dbgen.Agent) domain.Agent {
	return domain.Agent{
		ID:        row.ID,
		FlowID:    row.FlowID,
		Role:      row.Role,
		Model:     row.Model,
		Status:    domain.Status(row.Status),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func toDomainAgents(rows []dbgen.Agent) []domain.Agent {
	out := make([]domain.Agent, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainAgent(row))
	}
	return out
}

type executionRow interface {
	GetID() string
	GetFlowID() string
	GetActionID() string
	GetProfile() string
	GetRuntime() string
	GetMetadata() []byte
	GetStatus() string
	GetStartedAt() time.Time
	GetCompletedAt() sql.NullTime
}

type createExecutionRow dbgen.CreateExecutionRow

func (r createExecutionRow) GetID() string                { return r.ID }
func (r createExecutionRow) GetFlowID() string            { return r.FlowID }
func (r createExecutionRow) GetActionID() string          { return r.ActionID }
func (r createExecutionRow) GetProfile() string           { return r.Profile }
func (r createExecutionRow) GetRuntime() string           { return r.Runtime }
func (r createExecutionRow) GetMetadata() []byte          { return r.Metadata }
func (r createExecutionRow) GetStatus() string            { return r.Status }
func (r createExecutionRow) GetStartedAt() time.Time      { return r.StartedAt }
func (r createExecutionRow) GetCompletedAt() sql.NullTime { return r.CompletedAt }

type listExecutionRow dbgen.ListExecutionsByFlowRow

func (r listExecutionRow) GetID() string                { return r.ID }
func (r listExecutionRow) GetFlowID() string            { return r.FlowID }
func (r listExecutionRow) GetActionID() string          { return r.ActionID }
func (r listExecutionRow) GetProfile() string           { return r.Profile }
func (r listExecutionRow) GetRuntime() string           { return r.Runtime }
func (r listExecutionRow) GetMetadata() []byte          { return r.Metadata }
func (r listExecutionRow) GetStatus() string            { return r.Status }
func (r listExecutionRow) GetStartedAt() time.Time      { return r.StartedAt }
func (r listExecutionRow) GetCompletedAt() sql.NullTime { return r.CompletedAt }

func toDomainExecution(row executionRow) domain.Execution {
	return domain.Execution{
		ID:          row.GetID(),
		FlowID:      row.GetFlowID(),
		ActionID:    row.GetActionID(),
		Profile:     row.GetProfile(),
		Runtime:     row.GetRuntime(),
		Metadata:    toStringMap(row.GetMetadata()),
		Status:      domain.Status(row.GetStatus()),
		StartedAt:   row.GetStartedAt(),
		CompletedAt: nullTime(row.GetCompletedAt()),
	}
}

func toDomainExecutions(rows []dbgen.ListExecutionsByFlowRow) []domain.Execution {
	out := make([]domain.Execution, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainExecution(listExecutionRow(row)))
	}
	return out
}

func toDomainEvent(row dbgen.Event) domain.Event {
	return domain.Event{
		ID:        row.ID,
		Sequence:  row.Seq,
		FlowID:    row.FlowID,
		Type:      row.EventType,
		Message:   row.Message,
		Payload:   toAnyMap(row.Payload),
		CreatedAt: row.CreatedAt,
	}
}

func toDomainEvents(rows []dbgen.Event) []domain.Event {
	out := make([]domain.Event, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainEvent(row))
	}
	return out
}

func nullTime(value sql.NullTime) time.Time {
	if value.Valid {
		return value.Time
	}
	return time.Time{}
}
