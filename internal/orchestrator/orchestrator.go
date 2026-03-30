package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"nyx/internal/agentruntime"
	"nyx/internal/config"
	"nyx/internal/domain"
	"nyx/internal/functions"
	"nyx/internal/observability"
	"nyx/internal/queue"
	"nyx/internal/services/memory"
	"nyx/internal/store"
)

type Orchestrator struct {
	repo                 store.Repository
	gateway              *functions.Gateway
	pollInterval         time.Duration
	actionWait           time.Duration
	transport            queue.Transport
	runtime              *agentruntime.Runtime
	memory               *memory.Service
	requireRiskyApproval bool
	logger               *slog.Logger
	metrics              *observability.Registry
}

func New(repo store.Repository, gateway *functions.Gateway, pollInterval, actionWait time.Duration, transport queue.Transport, logger *slog.Logger, metrics *observability.Registry, requireRiskyApproval bool, runtimeOptions ...agentruntime.Option) *Orchestrator {
	if logger == nil {
		logger = observability.NewLogger(observability.LoggerConfig{Service: "nyx-orchestrator"})
	}
	if metrics == nil {
		metrics = observability.NewRegistry()
	}
	o := &Orchestrator{
		repo:                 repo,
		gateway:              gateway,
		pollInterval:         pollInterval,
		actionWait:           actionWait,
		transport:            transport,
		memory:               memory.New(repo),
		requireRiskyApproval: requireRiskyApproval,
		logger:               logger,
		metrics:              metrics,
	}
	o.runtime = agentruntime.New(repo, gateway.Definitions(), o.executeRuntimeAction, o.publish, append([]agentruntime.Option{agentruntime.WithLogger(logger)}, runtimeOptions...)...)
	return o
}

func (o *Orchestrator) RunForever(ctx context.Context) error {
	if o.transport != nil && o.transport.Mode() == config.TransportJetstream {
		return o.transport.ConsumeFlowRuns(ctx, o.handleFlowMessage)
	}

	return o.runPollLoop(ctx)
}

func (o *Orchestrator) runPollLoop(ctx context.Context) error {
	ticker := time.NewTicker(o.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		flow, ok, err := o.repo.ClaimNextQueuedFlow(ctx)
		if err != nil {
			return fmt.Errorf("claim next queued flow: %w", err)
		}
		if ok {
			o.metrics.IncCounter("nyx_orchestrator_flows_claimed_total", map[string]string{"mode": "poll"}, 1)
			o.runFlow(ctx, flow)
			continue
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (o *Orchestrator) handleFlowMessage(ctx context.Context, msg queue.FlowRunMessage) error {
	flow, err := o.repo.GetFlow(ctx, msg.FlowID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("get flow %s: %w", msg.FlowID, err)
	}

	switch flow.Status {
	case domain.StatusQueued:
		flow, err = o.repo.UpdateFlowStatus(ctx, flow.ID, domain.StatusRunning)
		if err != nil {
			return fmt.Errorf("update flow status to running: %w", err)
		}
	case domain.StatusRunning, domain.StatusPending:
	default:
		return nil
	}

	o.metrics.IncCounter("nyx_orchestrator_flows_claimed_total", map[string]string{"mode": o.transport.Mode()}, 1)
	o.runFlow(ctx, flow)
	return nil
}

func (o *Orchestrator) runFlow(ctx context.Context, flow domain.Flow) {
	o.logger.Info("flow execution started", "flow_id", flow.ID, "tenant_id", flow.TenantID, "status", flow.Status)
	if err := o.runtime.RunFlow(ctx, flow); err != nil {
		o.logger.Warn("failed to run flow", "flow_id", flow.ID, "err", err)
	}
	o.logger.Info("flow execution completed", "flow_id", flow.ID)
}

func (o *Orchestrator) publish(ctx context.Context, flowID, eventType, message string, payload map[string]any) {
	event, err := o.repo.RecordEvent(ctx, flowID, eventType, message, payload)
	if err != nil {
		o.logger.Warn("failed to record event", "flow_id", flowID, "event_type", eventType, "err", err)
		return
	}
	o.metrics.IncCounter("nyx_orchestrator_events_total", map[string]string{"type": eventType}, 1)
	if o.transport == nil {
		return
	}
	pubCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := o.transport.PublishEvent(pubCtx, queue.EventMessage{
		EventID:    event.ID,
		Sequence:   event.Sequence,
		FlowID:     event.FlowID,
		Type:       event.Type,
		Message:    event.Message,
		Payload:    event.Payload,
		OccurredAt: event.CreatedAt,
	}); err != nil {
		o.logger.Warn("failed to publish event", "flow_id", event.FlowID, "event_type", event.Type, "err", err)
	}
}
