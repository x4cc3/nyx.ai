package queue

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"nyx/internal/config"
)

func TestNoopTransportMode(t *testing.T) {
	transport := NewNoopTransport()
	if got := transport.Mode(); got != "poll" {
		t.Fatalf("expected poll mode, got %s", got)
	}
}

func TestNoopTransportClose(t *testing.T) {
	transport := NewNoopTransport()
	if err := transport.Close(); err != nil {
		t.Fatalf("expected nil close error, got %v", err)
	}
}

func TestNoopTransportPublishFlowRunSucceeds(t *testing.T) {
	transport := NewNoopTransport()
	err := transport.PublishFlowRun(context.Background(), FlowRunMessage{
		FlowID:   "flow-1",
		QueuedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestNoopTransportPublishEventSucceeds(t *testing.T) {
	transport := NewNoopTransport()
	err := transport.PublishEvent(context.Background(), EventMessage{
		FlowID:  "flow-1",
		Type:    "test.event",
		Message: "test",
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestNoopTransportPublishDeadLetterSucceeds(t *testing.T) {
	transport := NewNoopTransport()
	err := transport.PublishDeadLetter(context.Background(), DeadLetterMessage{
		Kind:   "test",
		Reason: "test failure",
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestNoopTransportConsumeFlowRunsReturnsUnsupported(t *testing.T) {
	transport := NewNoopTransport()
	err := transport.ConsumeFlowRuns(context.Background(), func(context.Context, FlowRunMessage) error {
		return nil
	})
	if !errors.Is(err, ErrTransportUnsupported) {
		t.Fatalf("expected ErrTransportUnsupported, got %v", err)
	}
}

func TestNoopTransportDispatchActionReturnsUnsupported(t *testing.T) {
	transport := NewNoopTransport()
	_, err := transport.DispatchAction(context.Background(), ActionRequestMessage{
		FlowID:       "flow-1",
		FunctionName: "terminal_exec",
	})
	if !errors.Is(err, ErrTransportUnsupported) {
		t.Fatalf("expected ErrTransportUnsupported, got %v", err)
	}
}

func TestNoopTransportConsumeActionRequestsReturnsUnsupported(t *testing.T) {
	transport := NewNoopTransport()
	err := transport.ConsumeActionRequests(context.Background(), func(context.Context, ActionRequestMessage) (ActionResultMessage, error) {
		return ActionResultMessage{}, nil
	})
	if !errors.Is(err, ErrTransportUnsupported) {
		t.Fatalf("expected ErrTransportUnsupported, got %v", err)
	}
}

func TestNoopTransportSatisfiesInterface(t *testing.T) {
	var _ Transport = NewNoopTransport()
}

func TestFlowRunMessageRoundTrip(t *testing.T) {
	msg := FlowRunMessage{
		FlowID:    "flow-123",
		QueuedAt:  time.Now().Truncate(time.Millisecond),
		QueueMode: "jetstream",
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded FlowRunMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.FlowID != msg.FlowID {
		t.Fatalf("expected flow_id %s, got %s", msg.FlowID, decoded.FlowID)
	}
	if decoded.QueueMode != "jetstream" {
		t.Fatalf("expected queue_mode jetstream, got %s", decoded.QueueMode)
	}
}

func TestActionRequestMessageRoundTrip(t *testing.T) {
	msg := ActionRequestMessage{
		FlowID:        "flow-1",
		TaskID:        "task-1",
		SubtaskID:     "sub-1",
		ActionID:      "action-1",
		AgentRole:     "executor",
		FunctionName:  "terminal_exec",
		ExecutionMode: "ephemeral",
		Input:         map[string]string{"command": "nmap -sV target"},
		RequestedAt:   time.Now().Truncate(time.Millisecond),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded ActionRequestMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.FunctionName != "terminal_exec" {
		t.Fatalf("expected terminal_exec, got %s", decoded.FunctionName)
	}
	if decoded.Input["command"] != "nmap -sV target" {
		t.Fatalf("expected command input, got %v", decoded.Input)
	}
}

func TestActionResultMessageRoundTrip(t *testing.T) {
	msg := ActionResultMessage{
		FlowID:       "flow-1",
		ActionID:     "action-1",
		FunctionName: "terminal_exec",
		Profile:      "pentest",
		Runtime:      "docker",
		Output:       map[string]string{"stdout": "results"},
		CompletedAt:  time.Now().Truncate(time.Millisecond),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded ActionResultMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Output["stdout"] != "results" {
		t.Fatalf("expected stdout output, got %v", decoded.Output)
	}
	if decoded.Error != "" {
		t.Fatalf("expected empty error, got %s", decoded.Error)
	}
}

func TestActionResultMessageWithError(t *testing.T) {
	msg := ActionResultMessage{
		FlowID:   "flow-1",
		ActionID: "action-1",
		Error:    "connection refused",
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded ActionResultMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Error != "connection refused" {
		t.Fatalf("expected error field, got %q", decoded.Error)
	}
}

func TestEventMessageRoundTrip(t *testing.T) {
	msg := EventMessage{
		EventID:    "evt-1",
		Sequence:   42,
		FlowID:     "flow-1",
		Type:       "flow.started",
		Message:    "Flow started",
		Payload:    map[string]any{"status": "running"},
		OccurredAt: time.Now().Truncate(time.Millisecond),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded EventMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Sequence != 42 {
		t.Fatalf("expected sequence 42, got %d", decoded.Sequence)
	}
	if decoded.Type != "flow.started" {
		t.Fatalf("expected flow.started, got %s", decoded.Type)
	}
}

func TestDeadLetterMessageRoundTrip(t *testing.T) {
	payload := json.RawMessage(`{"flow_id":"flow-1"}`)
	msg := DeadLetterMessage{
		Kind:     "action.request",
		Subject:  "nyx.actions.request",
		Reason:   "max retries exceeded",
		Error:    "timeout",
		Attempts: 3,
		Payload:  payload,
		FailedAt: time.Now().Truncate(time.Millisecond),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded DeadLetterMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", decoded.Attempts)
	}
	if decoded.Kind != "action.request" {
		t.Fatalf("expected action.request kind, got %s", decoded.Kind)
	}
}

func TestOpenTransportReturnsNoopWhenNoNATSURL(t *testing.T) {
	// config with empty NATS URL should return noop transport
	cfg := config.Config{}
	transport, err := OpenTransport(context.Background(), cfg)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if transport.Mode() != "poll" {
		t.Fatalf("expected poll mode for noop transport, got %s", transport.Mode())
	}
	transport.Close()
}
