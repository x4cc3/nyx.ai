package queue

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"nyx/internal/config"
)

func integrationNATSURL(t *testing.T) string {
	t.Helper()
	if value := os.Getenv("NYX_TEST_NATS_URL"); value != "" {
		return value
	}
	t.Skip("NYX_TEST_NATS_URL is not set")
	return ""
}

func newIntegrationTransport(t *testing.T, suffix string, maxDeliver int) *JetStreamTransport {
	t.Helper()

	cfg := config.Config{
		NATSURL:             integrationNATSURL(t),
		ServiceName:         "nyx-queue-test",
		FlowStream:          "NYX_FLOW_RUNS_" + suffix,
		FlowSubject:         "nyx.flows.run." + suffix,
		FlowConsumer:        "nyx-orchestrator-" + suffix,
		ActionStream:        "NYX_ACTION_REQUESTS_" + suffix,
		ActionSubject:       "nyx.actions.execute." + suffix,
		ActionConsumer:      "nyx-executor-" + suffix,
		ActionResultStream:  "NYX_ACTION_RESULTS_" + suffix,
		ActionResultSubject: "nyx.actions.result." + suffix,
		EventStream:         "NYX_EVENTS_" + suffix,
		EventSubject:        "nyx.events.flow." + suffix,
		DLQStream:           "NYX_DLQ_" + suffix,
		DLQSubject:          "nyx.dlq." + suffix,
		QueueMaxDeliver:     maxDeliver,
	}

	transport, err := OpenTransport(context.Background(), cfg)
	if err != nil {
		t.Fatalf("OpenTransport: %v", err)
	}
	t.Cleanup(func() { _ = transport.Close() })
	return transport.(*JetStreamTransport)
}

func TestJetStreamTransportFlowRoundTrip(t *testing.T) {
	transport := newIntegrationTransport(t, "flowroundtrip", 2)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	received := make(chan FlowRunMessage, 1)
	go func() {
		err := transport.ConsumeFlowRuns(ctx, func(_ context.Context, msg FlowRunMessage) error {
			received <- msg
			cancel()
			return nil
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Errorf("ConsumeFlowRuns: %v", err)
		}
	}()

	if err := transport.PublishFlowRun(context.Background(), FlowRunMessage{FlowID: "flow-1"}); err != nil {
		t.Fatalf("PublishFlowRun: %v", err)
	}

	select {
	case msg := <-received:
		if msg.FlowID != "flow-1" {
			t.Fatalf("unexpected flow id: %q", msg.FlowID)
		}
		if msg.QueueMode != "jetstream" {
			t.Fatalf("expected queue mode to be defaulted, got %q", msg.QueueMode)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for flow run")
	}
}

func TestJetStreamTransportDispatchAction(t *testing.T) {
	transport := newIntegrationTransport(t, "actiondispatch", 2)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		err := transport.ConsumeActionRequests(ctx, func(_ context.Context, msg ActionRequestMessage) (ActionResultMessage, error) {
			cancel()
			return ActionResultMessage{
				ActionID:     msg.ActionID,
				FlowID:       msg.FlowID,
				FunctionName: msg.FunctionName,
				Output:       map[string]string{"summary": "completed"},
			}, nil
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Errorf("ConsumeActionRequests: %v", err)
		}
	}()

	result, err := transport.DispatchAction(context.Background(), ActionRequestMessage{
		ActionID:     "action-1",
		FlowID:       "flow-1",
		FunctionName: "terminal_exec",
	})
	if err != nil {
		t.Fatalf("DispatchAction: %v", err)
	}
	if result.ActionID != "action-1" || result.Output["summary"] != "completed" {
		t.Fatalf("unexpected action result: %+v", result)
	}
}

func TestJetStreamTransportDeadLettersMalformedPayloads(t *testing.T) {
	transport := newIntegrationTransport(t, "malformed", 2)

	dlqSub, err := transport.nc.SubscribeSync(transport.deadLetterSubjectFor("flow_run"))
	if err != nil {
		t.Fatalf("SubscribeSync: %v", err)
	}
	defer func() { _ = dlqSub.Unsubscribe() }()

	if _, err := transport.js.Publish(transport.flowSubject, []byte("{")); err != nil {
		t.Fatalf("publish malformed flow: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_ = transport.ConsumeFlowRuns(ctx, func(context.Context, FlowRunMessage) error {
			t.Error("handler should not receive malformed payload")
			cancel()
			return nil
		})
	}()

	msg, err := dlqSub.NextMsg(5 * time.Second)
	if err != nil {
		t.Fatalf("NextMsg: %v", err)
	}
	cancel()

	var payload DeadLetterMessage
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		t.Fatalf("decode dead letter payload: %v", err)
	}
	if payload.Reason != "invalid_json" {
		t.Fatalf("unexpected dead letter reason: %+v", payload)
	}
}

func TestJetStreamTransportDeadLettersAfterMaxDeliver(t *testing.T) {
	transport := newIntegrationTransport(t, "maxdeliver", 1)

	dlqSub, err := transport.nc.SubscribeSync(transport.deadLetterSubjectFor("flow_run"))
	if err != nil {
		t.Fatalf("SubscribeSync: %v", err)
	}
	defer func() { _ = dlqSub.Unsubscribe() }()

	if err := transport.PublishFlowRun(context.Background(), FlowRunMessage{FlowID: "flow-1"}); err != nil {
		t.Fatalf("PublishFlowRun: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_ = transport.ConsumeFlowRuns(ctx, func(context.Context, FlowRunMessage) error {
			return errors.New("boom")
		})
	}()

	msg, err := dlqSub.NextMsg(5 * time.Second)
	if err != nil {
		t.Fatalf("NextMsg: %v", err)
	}
	cancel()

	var payload DeadLetterMessage
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		t.Fatalf("decode dead letter payload: %v", err)
	}
	if payload.Reason != "max_deliver_exceeded" || payload.Attempts != 1 {
		t.Fatalf("unexpected dead letter payload: %+v", payload)
	}
}
