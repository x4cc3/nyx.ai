package queue

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var ErrTransportUnsupported = errors.New("queue transport unsupported")

// NoopTransportMode is the mode string returned by NoopTransport.
const NoopTransportMode = "poll"

type FlowRunMessage struct {
	FlowID    string    `json:"flow_id"`
	QueuedAt  time.Time `json:"queued_at"`
	QueueMode string    `json:"queue_mode"`
}

type ActionRequestMessage struct {
	FlowID        string            `json:"flow_id"`
	TaskID        string            `json:"task_id"`
	SubtaskID     string            `json:"subtask_id"`
	ActionID      string            `json:"action_id"`
	AgentRole     string            `json:"agent_role"`
	FunctionName  string            `json:"function_name"`
	ExecutionMode string            `json:"execution_mode"`
	Input         map[string]string `json:"input"`
	RequestedAt   time.Time         `json:"requested_at"`
	QueueMode     string            `json:"queue_mode"`
}

type ActionResultMessage struct {
	FlowID       string            `json:"flow_id"`
	ActionID     string            `json:"action_id"`
	FunctionName string            `json:"function_name"`
	Profile      string            `json:"profile"`
	Runtime      string            `json:"runtime"`
	Output       map[string]string `json:"output"`
	Error        string            `json:"error,omitempty"`
	CompletedAt  time.Time         `json:"completed_at"`
	QueueMode    string            `json:"queue_mode"`
}

type EventMessage struct {
	EventID    string         `json:"event_id"`
	Sequence   int64          `json:"sequence"`
	FlowID     string         `json:"flow_id"`
	Type       string         `json:"type"`
	Message    string         `json:"message"`
	Payload    map[string]any `json:"payload"`
	OccurredAt time.Time      `json:"occurred_at"`
	QueueMode  string         `json:"queue_mode"`
}

type DeadLetterMessage struct {
	Kind      string          `json:"kind"`
	Subject   string          `json:"subject"`
	Reason    string          `json:"reason"`
	Error     string          `json:"error,omitempty"`
	Attempts  uint64          `json:"attempts"`
	Payload   json.RawMessage `json:"payload"`
	FailedAt  time.Time       `json:"failed_at"`
	QueueMode string          `json:"queue_mode"`
}

type Transport interface {
	PublishFlowRun(context.Context, FlowRunMessage) error
	ConsumeFlowRuns(context.Context, func(context.Context, FlowRunMessage) error) error
	DispatchAction(context.Context, ActionRequestMessage) (ActionResultMessage, error)
	ConsumeActionRequests(context.Context, func(context.Context, ActionRequestMessage) (ActionResultMessage, error)) error
	PublishEvent(context.Context, EventMessage) error
	PublishDeadLetter(context.Context, DeadLetterMessage) error
	Mode() string
	Close() error
}

type NoopTransport struct{}

func NewNoopTransport() *NoopTransport {
	return &NoopTransport{}
}

func (t *NoopTransport) PublishFlowRun(context.Context, FlowRunMessage) error { return nil }
func (t *NoopTransport) ConsumeFlowRuns(context.Context, func(context.Context, FlowRunMessage) error) error {
	return ErrTransportUnsupported
}
func (t *NoopTransport) DispatchAction(context.Context, ActionRequestMessage) (ActionResultMessage, error) {
	return ActionResultMessage{}, ErrTransportUnsupported
}
func (t *NoopTransport) ConsumeActionRequests(context.Context, func(context.Context, ActionRequestMessage) (ActionResultMessage, error)) error {
	return ErrTransportUnsupported
}
func (t *NoopTransport) PublishEvent(context.Context, EventMessage) error           { return nil }
func (t *NoopTransport) PublishDeadLetter(context.Context, DeadLetterMessage) error { return nil }
func (t *NoopTransport) Mode() string                                               { return NoopTransportMode }
func (t *NoopTransport) Close() error                                               { return nil }
