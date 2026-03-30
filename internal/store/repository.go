package store

import (
	"context"
	"errors"
	"time"

	"nyx/internal/domain"
)

var (
	ErrNotFound          = errors.New("not found")
	ErrInvalidPageCursor = errors.New("invalid page cursor")
)

// FlowReader provides read-only access to flows and flow details.
type FlowReader interface {
	GetFlow(ctx context.Context, id string) (domain.Flow, error)
	GetFlowForTenant(ctx context.Context, tenantID, id string) (domain.Flow, error)
	ListFlows(ctx context.Context) ([]domain.Flow, error)
	ListFlowsByTenant(ctx context.Context, tenantID string) ([]domain.Flow, error)
	ListFlowsPageByTenant(ctx context.Context, tenantID, afterID string, limit int) ([]domain.Flow, string, bool, error)
	FlowDetail(ctx context.Context, id string) (domain.FlowDetail, error)
	FlowDetailForTenant(ctx context.Context, tenantID, id string) (domain.FlowDetail, error)
}

// FlowWriter provides write and state-transition operations on flows.
type FlowWriter interface {
	CreateFlow(ctx context.Context, input domain.CreateFlowInput) (domain.Flow, error)
	CreateFlowForTenant(ctx context.Context, tenantID string, input domain.CreateFlowInput) (domain.Flow, error)
	QueueFlow(ctx context.Context, id string) (domain.Flow, error)
	QueueFlowForTenant(ctx context.Context, tenantID, id string) (domain.Flow, error)
	UpdateFlowStatus(ctx context.Context, id string, status domain.Status) (domain.Flow, error)
	ClaimNextQueuedFlow(ctx context.Context) (domain.Flow, bool, error)
}

// WorkUnitWriter manages the lifecycle of agents, tasks, subtasks, actions, and executions.
type WorkUnitWriter interface {
	CreateAgent(ctx context.Context, flowID, role, model string) (domain.Agent, error)
	UpdateAgentStatus(ctx context.Context, id string, status domain.Status) error
	CompleteAgent(ctx context.Context, id string, status domain.Status) error
	CreateTask(ctx context.Context, flowID, name, description, role string) (domain.Task, error)
	UpdateTaskStatus(ctx context.Context, id string, status domain.Status) error
	CompleteTask(ctx context.Context, id string, status domain.Status) error
	CreateSubtask(ctx context.Context, flowID, taskID, name, description, role string) (domain.Subtask, error)
	UpdateSubtaskStatus(ctx context.Context, id string, status domain.Status) error
	CompleteSubtask(ctx context.Context, id string, status domain.Status) error
	CreateAction(ctx context.Context, flowID, taskID, subtaskID, role, functionName, executionMode string, input map[string]string) (domain.Action, error)
	CompleteAction(ctx context.Context, id string, status domain.Status, output map[string]string) (domain.Action, error)
	CreateExecution(ctx context.Context, flowID, actionID, profile, runtime string, metadata map[string]string) (domain.Execution, error)
	CompleteExecution(ctx context.Context, id string, status domain.Status, profile, runtime string, metadata map[string]string) error
}

// ArtifactWriter persists artifacts and findings produced during flow execution.
type ArtifactWriter interface {
	AddArtifact(ctx context.Context, flowID, actionID, kind, name, content string, metadata map[string]string) (domain.Artifact, error)
	AddFinding(ctx context.Context, flowID, title, severity, description string) (domain.Finding, error)
}

// MemoryReadWriter stores and searches semantic memory entries.
type MemoryReadWriter interface {
	AddMemory(ctx context.Context, flowID, actionID, kind, content string, metadata map[string]string) (domain.Memory, error)
	SearchMemories(ctx context.Context, flowID, query string) ([]domain.Memory, error)
}

// ApprovalStore manages the approval lifecycle for risky operations.
type ApprovalStore interface {
	CreateApproval(ctx context.Context, flowID, tenantID, kind, requestedBy, reason string, payload map[string]string) (domain.Approval, error)
	GetApproval(ctx context.Context, id string) (domain.Approval, error)
	ListApprovalsByTenant(ctx context.Context, tenantID string) ([]domain.Approval, error)
	ListApprovalsPageByTenant(ctx context.Context, tenantID, afterID string, limit int) ([]domain.Approval, string, bool, error)
	ListApprovalsByFlow(ctx context.Context, flowID string) ([]domain.Approval, error)
	ReviewApproval(ctx context.Context, id string, approved bool, reviewedBy, note string) (domain.Approval, error)
}

// EventStore records and retrieves flow lifecycle events.
type EventStore interface {
	RecordEvent(ctx context.Context, flowID, eventType, message string, payload map[string]any) (domain.Event, error)
	ListEvents(ctx context.Context, flowID string, afterSequence int64) ([]domain.Event, error)
}

// Lifecycle manages store initialization and teardown.
type Lifecycle interface {
	Init(context.Context) error
	Ping(context.Context) error
	Close() error
}

// Repository composes all store sub-interfaces into a single aggregate.
// Callers should prefer accepting the narrowest sub-interface they need.
type Repository interface {
	Lifecycle
	FlowReader
	FlowWriter
	WorkUnitWriter
	ArtifactWriter
	MemoryReadWriter
	ApprovalStore
	EventStore
}

type EventPoller interface {
	NextSequence() int64
	SetSequence(int64)
	LastSeen() time.Time
	Touch()
}
