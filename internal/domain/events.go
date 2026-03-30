package domain

// EventType represents the type of a domain event recorded during flow execution.
type EventType = string

const (
	EventFlowStatus                   EventType = "flow.status"
	EventFlowCreated                  EventType = "flow.created"
	EventFlowQueued                   EventType = "flow.queued"
	EventFlowDispatched               EventType = "flow.dispatched"
	EventFlowDispatchFailed           EventType = "flow.dispatch_failed"
	EventFlowCancelled                EventType = "flow.cancelled"
	EventFlowApprovalRequested        EventType = "flow.approval_requested"
	EventFlowApprovalApproved         EventType = "flow.approval_approved"
	EventFlowApprovalRejected         EventType = "flow.approval_rejected"
	EventRuntimePlanReady             EventType = "runtime.plan.ready"
	EventRuntimePlanGenerated         EventType = "runtime.plan.generated"
	EventActionStarted                EventType = "action.started"
	EventActionCompleted              EventType = "action.completed"
	EventActionFailed                 EventType = "action.failed"
	EventActionApprovalRequested      EventType = "action.approval_requested"
	EventActionApprovalVerified       EventType = "action.approval_verified"
	EventActionDecided                EventType = "action.decided"
	EventActionRepetitionBlocked      EventType = "action.repetition_blocked"
	EventAgentStarted                 EventType = "agent.started"
	EventAgentCompleted               EventType = "agent.completed"
	EventAgentContext                 EventType = "agent.context"
	EventTaskCreated                  EventType = "task.created"
	EventTaskReassigned               EventType = "task.reassigned"
	EventSubtaskStarted               EventType = "subtask.started"
	EventSubtaskAttempt               EventType = "subtask.attempt"
	EventSubtaskCompleted             EventType = "subtask.completed"
	EventSubtaskRetry                 EventType = "subtask.retry"
	EventSubtaskFailed                EventType = "subtask.failed"
	EventSubtaskEscalated             EventType = "subtask.escalated"
	EventSubtaskOperatorInputRequired EventType = "subtask.operator_input_required"
)

// ApprovalKind identifies the category of an approval request.
type ApprovalKind = string

const (
	ApprovalKindFlowStart  ApprovalKind = "flow.start"
	ApprovalKindRiskReview ApprovalKind = "action.risk_review"
)

// ApprovalStatus represents the state of an approval.
type ApprovalStatus = string

const (
	ApprovalStatusPending  ApprovalStatus = "pending"
	ApprovalStatusApproved ApprovalStatus = "approved"
	ApprovalStatusRejected ApprovalStatus = "rejected"
)
