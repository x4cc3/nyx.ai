package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"nyx/internal/config"

	"github.com/nats-io/nats.go"
)

type JetStreamTransport struct {
	nc                  *nats.Conn
	js                  nats.JetStreamContext
	mode                string
	flowStream          string
	flowSubject         string
	flowConsumer        string
	actionStream        string
	actionSubject       string
	actionConsumer      string
	actionResultStream  string
	actionResultSubject string
	eventStream         string
	eventSubject        string
	dlqStream           string
	dlqSubject          string
	maxDeliver          int
}

func OpenTransport(ctx context.Context, cfg config.Config) (Transport, error) {
	if cfg.NATSURL == "" {
		return NewNoopTransport(), nil
	}

	nc, err := nats.Connect(cfg.NATSURL, nats.Name(cfg.ServiceName))
	if err != nil {
		return nil, err
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, err
	}

	t := &JetStreamTransport{
		nc:                  nc,
		js:                  js,
		mode:                "jetstream",
		flowStream:          cfg.FlowStream,
		flowSubject:         cfg.FlowSubject,
		flowConsumer:        cfg.FlowConsumer,
		actionStream:        cfg.ActionStream,
		actionSubject:       cfg.ActionSubject,
		actionConsumer:      cfg.ActionConsumer,
		actionResultStream:  cfg.ActionResultStream,
		actionResultSubject: cfg.ActionResultSubject,
		eventStream:         cfg.EventStream,
		eventSubject:        cfg.EventSubject,
		dlqStream:           cfg.DLQStream,
		dlqSubject:          cfg.DLQSubject,
		maxDeliver:          cfg.QueueMaxDeliver,
	}
	if t.maxDeliver < 1 {
		t.maxDeliver = 1
	}

	if err := t.ensureInfrastructure(ctx); err != nil {
		nc.Close()
		return nil, err
	}

	return t, nil
}

func (t *JetStreamTransport) PublishFlowRun(ctx context.Context, msg FlowRunMessage) error {
	if msg.QueueMode == "" {
		msg.QueueMode = t.mode
	}
	if msg.QueuedAt.IsZero() {
		msg.QueuedAt = time.Now().UTC()
	}
	return t.publish(ctx, t.flowSubject, msg)
}

func (t *JetStreamTransport) ConsumeFlowRuns(ctx context.Context, handler func(context.Context, FlowRunMessage) error) error {
	sub, err := t.js.PullSubscribe(t.flowSubject, t.flowConsumer, nats.Bind(t.flowStream, t.flowConsumer))
	if err != nil {
		return err
	}
	defer func() { _ = sub.Unsubscribe() }()

	for {
		msgs, err := t.fetch(ctx, sub)
		if err != nil {
			return err
		}
		for _, msg := range msgs {
			var flowMsg FlowRunMessage
			if err := json.Unmarshal(msg.Data, &flowMsg); err != nil {
				if dlqErr := t.deadLetterMalformed(ctx, msg, "flow_run", err); dlqErr != nil {
					return dlqErr
				}
				continue
			}
			if err := handler(ctx, flowMsg); err != nil {
				if retryErr := t.retryOrDeadLetter(ctx, msg, "flow_run", err); retryErr != nil {
					return retryErr
				}
				continue
			}
			if err := msg.Ack(); err != nil {
				return err
			}
		}
	}
}

func (t *JetStreamTransport) DispatchAction(ctx context.Context, req ActionRequestMessage) (ActionResultMessage, error) {
	if req.QueueMode == "" {
		req.QueueMode = t.mode
	}
	if req.RequestedAt.IsZero() {
		req.RequestedAt = time.Now().UTC()
	}

	subject := t.actionResultSubjectFor(req.ActionID)
	sub, err := t.nc.SubscribeSync(subject)
	if err != nil {
		return ActionResultMessage{}, err
	}
	defer func() { _ = sub.Unsubscribe() }()

	if err := t.publish(ctx, t.actionSubject, req); err != nil {
		return ActionResultMessage{}, err
	}

	msg, err := sub.NextMsgWithContext(ctx)
	if err != nil {
		return ActionResultMessage{}, err
	}

	var result ActionResultMessage
	if err := json.Unmarshal(msg.Data, &result); err != nil {
		return ActionResultMessage{}, err
	}
	return result, nil
}

func (t *JetStreamTransport) ConsumeActionRequests(ctx context.Context, handler func(context.Context, ActionRequestMessage) (ActionResultMessage, error)) error {
	sub, err := t.js.PullSubscribe(t.actionSubject, t.actionConsumer, nats.Bind(t.actionStream, t.actionConsumer))
	if err != nil {
		return err
	}
	defer func() { _ = sub.Unsubscribe() }()

	for {
		msgs, err := t.fetch(ctx, sub)
		if err != nil {
			return err
		}
		for _, msg := range msgs {
			var req ActionRequestMessage
			if err := json.Unmarshal(msg.Data, &req); err != nil {
				if dlqErr := t.deadLetterMalformed(ctx, msg, "action_request", err); dlqErr != nil {
					return dlqErr
				}
				continue
			}

			result, err := handler(ctx, req)
			if err != nil {
				if retryErr := t.retryOrDeadLetter(ctx, msg, "action_request", err); retryErr != nil {
					return retryErr
				}
				continue
			}
			if result.ActionID == "" {
				result.ActionID = req.ActionID
			}
			if result.FlowID == "" {
				result.FlowID = req.FlowID
			}
			if result.FunctionName == "" {
				result.FunctionName = req.FunctionName
			}
			if result.QueueMode == "" {
				result.QueueMode = t.mode
			}
			if result.CompletedAt.IsZero() {
				result.CompletedAt = time.Now().UTC()
			}

			if err := t.publish(ctx, t.actionResultSubjectFor(req.ActionID), result); err != nil {
				if retryErr := t.retryOrDeadLetter(ctx, msg, "action_result_publish", err); retryErr != nil {
					return retryErr
				}
				continue
			}
			if err := msg.Ack(); err != nil {
				return err
			}
		}
	}
}

func (t *JetStreamTransport) PublishEvent(ctx context.Context, msg EventMessage) error {
	if msg.QueueMode == "" {
		msg.QueueMode = t.mode
	}
	if msg.OccurredAt.IsZero() {
		msg.OccurredAt = time.Now().UTC()
	}
	return t.publish(ctx, t.eventSubjectFor(msg.FlowID), msg)
}

func (t *JetStreamTransport) PublishDeadLetter(ctx context.Context, msg DeadLetterMessage) error {
	if msg.QueueMode == "" {
		msg.QueueMode = t.mode
	}
	if msg.FailedAt.IsZero() {
		msg.FailedAt = time.Now().UTC()
	}
	return t.publish(ctx, t.deadLetterSubjectFor(msg.Kind), msg)
}

func (t *JetStreamTransport) Mode() string { return t.mode }
func (t *JetStreamTransport) Close() error { t.nc.Close(); return nil }

func (t *JetStreamTransport) ensureInfrastructure(ctx context.Context) error {
	if err := t.ensureStream(ctx, t.flowStream, []string{t.flowSubject}, nats.WorkQueuePolicy); err != nil {
		return err
	}
	if err := t.ensureStream(ctx, t.actionStream, []string{t.actionSubject}, nats.WorkQueuePolicy); err != nil {
		return err
	}
	if err := t.ensureStream(ctx, t.actionResultStream, []string{t.actionResultSubject + ".*"}, nats.LimitsPolicy); err != nil {
		return err
	}
	if err := t.ensureStream(ctx, t.eventStream, []string{t.eventSubject + ".*"}, nats.LimitsPolicy); err != nil {
		return err
	}
	if err := t.ensureStream(ctx, t.dlqStream, []string{t.dlqSubject + ".*"}, nats.LimitsPolicy); err != nil {
		return err
	}
	if err := t.ensureConsumer(ctx, t.flowStream, t.flowConsumer, t.flowSubject); err != nil {
		return err
	}
	if err := t.ensureConsumer(ctx, t.actionStream, t.actionConsumer, t.actionSubject); err != nil {
		return err
	}
	return nil
}

func (t *JetStreamTransport) ensureStream(ctx context.Context, name string, subjects []string, retention nats.RetentionPolicy) error {
	_, err := t.js.StreamInfo(name, nats.Context(ctx))
	if err == nil {
		return nil
	}
	if !errors.Is(err, nats.ErrStreamNotFound) {
		return err
	}

	_, err = t.js.AddStream(&nats.StreamConfig{
		Name:      name,
		Subjects:  subjects,
		Retention: retention,
		Storage:   nats.FileStorage,
		MaxAge:    24 * time.Hour,
	}, nats.Context(ctx))
	if err != nil && !errors.Is(err, nats.ErrStreamNameAlreadyInUse) {
		return err
	}
	return nil
}

func (t *JetStreamTransport) ensureConsumer(ctx context.Context, stream, consumer, subject string) error {
	_, err := t.js.ConsumerInfo(stream, consumer, nats.Context(ctx))
	if err == nil {
		return nil
	}
	if !errors.Is(err, nats.ErrConsumerNotFound) {
		return err
	}

	_, err = t.js.AddConsumer(stream, &nats.ConsumerConfig{
		Durable:       consumer,
		AckPolicy:     nats.AckExplicitPolicy,
		AckWait:       60 * time.Second,
		FilterSubject: subject,
		MaxAckPending: 1,
		MaxWaiting:    64,
		MaxDeliver:    t.maxDeliver,
	}, nats.Context(ctx))
	if err != nil && !errors.Is(err, nats.ErrConsumerNameAlreadyInUse) {
		return err
	}
	return nil
}

func (t *JetStreamTransport) fetch(ctx context.Context, sub *nats.Subscription) ([]*nats.Msg, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		msgs, err := sub.Fetch(1, nats.MaxWait(time.Second))
		if err != nil {
			if errors.Is(err, nats.ErrTimeout) {
				continue
			}
			return nil, err
		}
		return msgs, nil
	}
}

func (t *JetStreamTransport) publish(ctx context.Context, subject string, payload any) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = t.js.Publish(subject, body, nats.Context(ctx))
	return err
}

func (t *JetStreamTransport) deadLetterMalformed(ctx context.Context, msg *nats.Msg, kind string, decodeErr error) error {
	dlqErr := t.PublishDeadLetter(ctx, DeadLetterMessage{
		Kind:     kind,
		Subject:  msg.Subject,
		Reason:   "invalid_json",
		Error:    decodeErr.Error(),
		Attempts: 1,
		Payload:  append(json.RawMessage(nil), msg.Data...),
	})
	if dlqErr != nil {
		return dlqErr
	}
	return msg.Term()
}

func (t *JetStreamTransport) retryOrDeadLetter(ctx context.Context, msg *nats.Msg, kind string, handlerErr error) error {
	meta, _ := msg.Metadata()
	attempts := uint64(1)
	if meta != nil && meta.NumDelivered > 0 {
		attempts = meta.NumDelivered
	}

	if attempts >= uint64(t.maxDeliver) {
		dlqErr := t.PublishDeadLetter(ctx, DeadLetterMessage{
			Kind:     kind,
			Subject:  msg.Subject,
			Reason:   "max_deliver_exceeded",
			Error:    handlerErr.Error(),
			Attempts: attempts,
			Payload:  append(json.RawMessage(nil), msg.Data...),
		})
		if dlqErr != nil {
			return dlqErr
		}
		return msg.Term()
	}

	return msg.Nak()
}

func (t *JetStreamTransport) actionResultSubjectFor(actionID string) string {
	return fmt.Sprintf("%s.%s", t.actionResultSubject, subjectToken(actionID))
}

func (t *JetStreamTransport) eventSubjectFor(flowID string) string {
	return fmt.Sprintf("%s.%s", t.eventSubject, subjectToken(flowID))
}

func (t *JetStreamTransport) deadLetterSubjectFor(kind string) string {
	return fmt.Sprintf("%s.%s", t.dlqSubject, subjectToken(kind))
}

func subjectToken(value string) string {
	replacer := strings.NewReplacer(".", "_", " ", "_", "*", "_", ">", "_")
	out := strings.Trim(replacer.Replace(value), " _")
	if out == "" {
		return "unknown"
	}
	return out
}
