package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"nyx/internal/domain"
	"nyx/internal/queue"
	"nyx/internal/reports"
	"nyx/internal/store"
)

func (s *Server) handleFlows(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ctx := r.Context()
		limit, err := parseListLimit(r.URL.Query().Get("limit"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_pagination", err.Error())
			return
		}
		after := strings.TrimSpace(r.URL.Query().Get("after"))
		flows, nextAfter, hasMore, err := s.repo.ListFlowsPageByTenant(ctx, currentTenant(r, s.cfg.DefaultTenant), after, limit)
		if err != nil {
			if errors.Is(err, store.ErrInvalidPageCursor) {
				writeError(w, http.StatusBadRequest, "invalid_pagination", err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, "list_flows_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"flows":     flows,
			"page_info": newPageInfo(limit, after, len(flows), nextAfter, hasMore),
		})
	case http.MethodPost:
		var input domain.CreateFlowInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
			return
		}
		if fieldErrors := validateCreateFlowInput(&input); len(fieldErrors) > 0 {
			writeErrorWithFields(w, http.StatusBadRequest, "invalid_flow", "Flow validation failed", fieldErrors)
			return
		}
		ctx := r.Context()
		flow, err := s.repo.CreateFlowForTenant(ctx, currentTenant(r, s.cfg.DefaultTenant), input)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "create_flow_failed", err.Error())
			return
		}
		s.fanoutEvent(flow.ID, domain.EventFlowCreated, "Flow created", map[string]any{"status": flow.Status})
		writeJSON(w, http.StatusCreated, flow)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleWorkspaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	ids := requestedIDs(r.URL.Query()["flow_ids"])
	if len(ids) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_flow_ids", "At least one flow id is required")
		return
	}

	tenantID := currentTenant(r, s.cfg.DefaultTenant)
	workspaces := make([]domain.Workspace, 0, len(ids))
	failures := make(map[string]string)
	for _, flowID := range ids {
		workspace, err := s.workspacePayload(r.Context(), tenantID, flowID)
		if err != nil {
			failures[flowID] = "Flow not found"
			continue
		}
		workspaces = append(workspaces, workspace)
	}

	payload := map[string]any{"workspaces": workspaces}
	if len(failures) > 0 {
		payload["errors"] = failures
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) handleFlowRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/flows/")
	path = strings.Trim(path, "/")
	if path == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	parts := strings.Split(path, "/")
	flowID := parts[0]

	if len(parts) == 1 {
		s.handleFlowDetail(w, r, flowID)
		return
	}

	switch parts[1] {
	case "start":
		s.handleFlowStart(w, r, flowID)
	case "cancel":
		s.handleFlowCancel(w, r, flowID)
	case "events":
		s.handleFlowEvents(w, r, flowID)
	case "report":
		s.handleFlowReport(w, r, flowID)
	case "tasks", "subtasks", "actions", "artifacts", "findings", "agents", "memories", "executions":
		s.handleFlowCollection(w, r, flowID, parts[1])
	case "approvals":
		s.handleFlowApprovals(w, r, flowID)
	case "workspace":
		s.handleFlowWorkspace(w, r, flowID)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (s *Server) handleFlowReport(w http.ResponseWriter, r *http.Request, flowID string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	workspace, err := s.workspacePayload(r.Context(), currentTenant(r, s.cfg.DefaultTenant), flowID)
	if err != nil {
		writeError(w, http.StatusNotFound, "flow_not_found", "Flow not found")
		return
	}
	doc := reports.Build(workspace)
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		format = "markdown"
	}
	s.metrics.IncCounter("nyx_reports_generated_total", map[string]string{"format": format}, 1)
	switch format {
	case "json":
		payload, err := doc.JSON()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "report_failed", err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	case "pdf":
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-report.pdf"`, flowID))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(doc.PDF())
	default:
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-report.md"`, flowID))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(doc.Markdown()))
	}
}

func (s *Server) handleFlowDetail(w http.ResponseWriter, r *http.Request, flowID string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	detail, err := s.repo.FlowDetailForTenant(ctx, currentTenant(r, s.cfg.DefaultTenant), flowID)
	if err != nil {
		writeError(w, http.StatusNotFound, "flow_not_found", "Flow not found")
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) handleFlowStart(w http.ResponseWriter, r *http.Request, flowID string) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	tenantID := currentTenant(r, s.cfg.DefaultTenant)
	flow, err := s.repo.GetFlowForTenant(ctx, tenantID, flowID)
	if err != nil {
		writeError(w, http.StatusNotFound, "flow_not_found", "Flow not found")
		return
	}
	switch flow.Status {
	case domain.StatusCancelled:
		writeError(w, http.StatusConflict, "flow_cancelled", "Cancelled flows cannot be started")
		return
	case domain.StatusCompleted, domain.StatusFailed:
		writeError(w, http.StatusConflict, "flow_closed", "Closed flows cannot be started")
		return
	}

	if s.cfg.RequireFlowApproval {
		existing, _ := s.repo.ListApprovalsByFlow(ctx, flowID)
		for _, approval := range existing {
			if approval.Kind == domain.ApprovalKindFlowStart && approval.Status == domain.ApprovalStatusPending {
				writeJSON(w, http.StatusAccepted, map[string]any{
					"status":      "approval_pending",
					"approval_id": approval.ID,
					"flow_id":     flowID,
				})
				return
			}
		}

		approval, err := s.repo.CreateApproval(ctx, flowID, tenantID, domain.ApprovalKindFlowStart, currentOperator(r), "Flow start requires operator approval before risky execution begins.", map[string]string{
			"flow_id": flowID,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "approval_create_failed", err.Error())
			return
		}
		s.recordAndFanoutEvent(ctx, flowID, domain.EventFlowApprovalRequested, "Flow start approval requested", map[string]any{
			"approval_id": approval.ID,
			"tenant_id":   tenantID,
		})
		writeJSON(w, http.StatusAccepted, map[string]any{
			"status":      "approval_pending",
			"approval_id": approval.ID,
			"flow_id":     flowID,
		})
		return
	}
	s.dispatchFlow(w, r, flowID, tenantID)
}

func (s *Server) handleFlowCancel(w http.ResponseWriter, r *http.Request, flowID string) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	tenantID := currentTenant(r, s.cfg.DefaultTenant)
	flow, err := s.repo.GetFlowForTenant(ctx, tenantID, flowID)
	if err != nil {
		writeError(w, http.StatusNotFound, "flow_not_found", "Flow not found")
		return
	}
	switch flow.Status {
	case domain.StatusCompleted, domain.StatusFailed:
		writeError(w, http.StatusConflict, "flow_closed", "Closed flows cannot be cancelled")
		return
	case domain.StatusCancelled:
		writeJSON(w, http.StatusOK, map[string]any{"status": flow.Status, "flow_id": flow.ID})
		return
	}

	updated, err := s.repo.UpdateFlowStatus(ctx, flowID, domain.StatusCancelled)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "flow_cancel_failed", err.Error())
		return
	}
	s.recordAndFanoutEvent(ctx, flowID, domain.EventFlowCancelled, "Flow cancelled by operator", map[string]any{
		"operator": currentOperator(r),
		"status":   updated.Status,
	})
	writeJSON(w, http.StatusOK, map[string]any{"status": updated.Status, "flow_id": updated.ID})
}

// recordAndFanoutEvent persists an event and publishes it for real-time consumers.
func (s *Server) recordAndFanoutEvent(ctx context.Context, flowID, eventType, message string, payload map[string]any) {
	_, _ = s.repo.RecordEvent(ctx, flowID, eventType, message, payload)
	s.fanoutEvent(flowID, eventType, message, payload)
}

func (s *Server) fanoutEvent(flowID, eventType, message string, payload map[string]any) {
	if s.queue == nil {
		return
	}
	pubCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = s.queue.PublishEvent(pubCtx, queue.EventMessage{
		FlowID:     flowID,
		Type:       eventType,
		Message:    message,
		Payload:    payload,
		OccurredAt: time.Now().UTC(),
	})
}

// approvalEventInfo returns the event type and message for an approval status.
func approvalEventInfo(status string) (eventType, message string) {
	if status == domain.ApprovalStatusApproved {
		return domain.EventFlowApprovalApproved, "Flow approval approved"
	}
	return domain.EventFlowApprovalRejected, "Flow approval rejected"
}

func (s *Server) handleFlowEvents(w http.ResponseWriter, r *http.Request, flowID string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	if _, err := s.repo.GetFlowForTenant(ctx, currentTenant(r, s.cfg.DefaultTenant), flowID); err != nil {
		writeError(w, http.StatusNotFound, "flow_not_found", "Flow not found")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "stream_unsupported", "Streaming not supported")
		return
	}

	detail, _ := s.repo.FlowDetailForTenant(ctx, currentTenant(r, s.cfg.DefaultTenant), flowID)
	writeSSE(w, "snapshot", detail)
	flusher.Flush()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	keepAlive := time.NewTicker(15 * time.Second)
	defer keepAlive.Stop()

	var afterSequence int64
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			events, err := s.repo.ListEvents(ctx, flowID, afterSequence)
			if err != nil {
				writeSSE(w, "error", map[string]string{"message": "failed to load events"})
				flusher.Flush()
				continue
			}
			for _, event := range events {
				afterSequence = event.Sequence
				writeSSE(w, event.Type, event)
			}
			flusher.Flush()
		case <-keepAlive.C:
			fmt.Fprint(w, ": keep-alive\n\n")
			flusher.Flush()
		}
	}
}

func (s *Server) handleFlowCollection(w http.ResponseWriter, r *http.Request, flowID, collection string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()
	detail, err := s.repo.FlowDetailForTenant(ctx, currentTenant(r, s.cfg.DefaultTenant), flowID)
	if err != nil {
		writeError(w, http.StatusNotFound, "flow_not_found", "Flow not found")
		return
	}
	payload := map[string]any{
		"flow_id": flowID,
	}
	switch collection {
	case "tasks":
		payload["tasks"] = detail.Tasks
	case "subtasks":
		payload["subtasks"] = detail.Subtasks
	case "actions":
		payload["actions"] = detail.Actions
	case "artifacts":
		payload["artifacts"] = detail.Artifacts
	case "findings":
		payload["findings"] = detail.Findings
	case "agents":
		payload["agents"] = detail.Agents
	case "memories":
		payload["memories"] = detail.Memories
	case "executions":
		payload["executions"] = detail.Executions
	default:
		w.WriteHeader(http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) handleFlowApprovals(w http.ResponseWriter, r *http.Request, flowID string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()
	flow, err := s.repo.GetFlowForTenant(ctx, currentTenant(r, s.cfg.DefaultTenant), flowID)
	if err != nil {
		writeError(w, http.StatusNotFound, "flow_not_found", "Flow not found")
		return
	}
	approvals, err := s.repo.ListApprovalsByFlow(ctx, flow.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list_approvals_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"approvals": approvals, "flow_id": flowID})
}

func (s *Server) handleFlowWorkspace(w http.ResponseWriter, r *http.Request, flowID string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	workspace, err := s.workspacePayload(r.Context(), currentTenant(r, s.cfg.DefaultTenant), flowID)
	if err != nil {
		writeError(w, http.StatusNotFound, "flow_not_found", "Flow not found")
		return
	}
	writeJSON(w, http.StatusOK, workspace)
}

func (s *Server) dispatchFlow(w http.ResponseWriter, r *http.Request, flowID, tenantID string) {
	ctx := r.Context()
	previous, err := s.repo.GetFlowForTenant(ctx, tenantID, flowID)
	if err != nil {
		writeError(w, http.StatusNotFound, "flow_not_found", "Flow not found")
		return
	}

	flow, err := s.repo.QueueFlowForTenant(ctx, tenantID, flowID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "flow_queue_failed", err.Error())
		return
	}
	s.recordAndFanoutEvent(ctx, flowID, domain.EventFlowQueued, "Flow queued for orchestrator pickup", map[string]any{"status": flow.Status})
	if err := s.queue.PublishFlowRun(r.Context(), queue.FlowRunMessage{FlowID: flowID}); err != nil {
		payload := map[string]any{
			"queue_mode": s.queue.Mode(),
			"error":      err.Error(),
		}
		if _, rollbackErr := s.repo.UpdateFlowStatus(ctx, flowID, previous.Status); rollbackErr != nil {
			payload["rollback_error"] = rollbackErr.Error()
			s.recordAndFanoutEvent(ctx, flowID, domain.EventFlowDispatchFailed, "Flow dispatch failed", payload)
			writeError(w, http.StatusInternalServerError, "flow_dispatch_failed", fmt.Sprintf("%s (status rollback failed: %v)", err, rollbackErr))
			return
		}
		payload["restored_status"] = previous.Status
		s.recordAndFanoutEvent(ctx, flowID, domain.EventFlowDispatchFailed, "Flow dispatch failed", payload)
		writeError(w, http.StatusInternalServerError, "flow_dispatch_failed", err.Error())
		return
	}
	s.recordAndFanoutEvent(ctx, flowID, domain.EventFlowDispatched, "Flow dispatched to execution transport", map[string]any{
		"queue_mode": s.queue.Mode(),
	})
	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":     "queued",
		"flow_id":    flowID,
		"queue_mode": s.queue.Mode(),
	})
}

func (s *Server) workspacePayload(ctx context.Context, tenantID, flowID string) (domain.Workspace, error) {
	detail, err := s.repo.FlowDetailForTenant(ctx, tenantID, flowID)
	if err != nil {
		return domain.Workspace{}, err
	}
	approvals, err := s.repo.ListApprovalsByFlow(ctx, flowID)
	if err != nil {
		return domain.Workspace{}, err
	}
	needsReview := false
	for _, approval := range approvals {
		if approval.Status == domain.ApprovalStatusPending {
			needsReview = true
			break
		}
	}
	return domain.Workspace{
		Flow:                     detail.Flow,
		Tasks:                    detail.Tasks,
		Subtasks:                 detail.Subtasks,
		Actions:                  detail.Actions,
		Artifacts:                detail.Artifacts,
		Memories:                 detail.Memories,
		Findings:                 detail.Findings,
		Agents:                   detail.Agents,
		Executions:               detail.Executions,
		Approvals:                approvals,
		Functions:                s.gateway.Definitions(),
		TenantID:                 tenantID,
		QueueMode:                s.queue.Mode(),
		ExecutorMode:             s.cfg.ExecutorMode,
		BrowserMode:              s.browserMode(),
		ExecutorNetworkMode:      s.executorNetworkMode(),
		ExecutorNetworkName:      s.executorNetworkName(),
		ExecutorNetRawEnabled:    s.cfg.ExecutorEnableNetRaw,
		TerminalNetworkEnabled:   s.terminalNetworkEnabled(),
		RiskyApprovalRequired:    s.cfg.RequireRiskyApproval,
		FlowMaxConcurrentActions: s.cfg.FlowMaxConcurrentActions,
		FlowMinActionIntervalMS:  s.cfg.FlowMinActionInterval.Milliseconds(),
		BrowserWarning:           s.browserWarning(),
		NetworkWarning:           s.networkWarning(),
		RiskWarning:              s.riskWarning(),
		NeedsReview:              needsReview,
	}, nil
}

func validateCreateFlowInput(input *domain.CreateFlowInput) map[string]string {
	input.Name = strings.TrimSpace(input.Name)
	input.Target = strings.TrimSpace(input.Target)
	input.Objective = strings.TrimSpace(input.Objective)

	fieldErrors := make(map[string]string)
	if input.Name == "" {
		fieldErrors["name"] = "Flow name is required."
	}
	if input.Target == "" {
		fieldErrors["target"] = "Target URL is required."
	} else if parsed, err := url.Parse(input.Target); err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || strings.TrimSpace(parsed.Host) == "" {
		fieldErrors["target"] = "Target must be a valid http(s) URL."
	}
	if input.Objective == "" {
		fieldErrors["objective"] = "Objective is required."
	}
	if len(input.Name) > 160 {
		fieldErrors["name"] = "Flow name must be 160 characters or fewer."
	}
	if len(input.Target) > 2048 {
		fieldErrors["target"] = "Target must be 2048 characters or fewer."
	}
	if len(input.Objective) > 4000 {
		fieldErrors["objective"] = "Objective must be 4000 characters or fewer."
	}
	return fieldErrors
}
