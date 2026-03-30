package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"nyx/internal/domain"
	"nyx/internal/store"
)

func (s *Server) handleApprovals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()
	limit, err := parseListLimit(r.URL.Query().Get("limit"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_pagination", err.Error())
		return
	}
	after := strings.TrimSpace(r.URL.Query().Get("after"))
	approvals, nextAfter, hasMore, err := s.repo.ListApprovalsPageByTenant(ctx, currentTenant(r, s.cfg.DefaultTenant), after, limit)
	if err != nil {
		if errors.Is(err, store.ErrInvalidPageCursor) {
			writeError(w, http.StatusBadRequest, "invalid_pagination", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "list_approvals_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"approvals": approvals,
		"page_info": newPageInfo(limit, after, len(approvals), nextAfter, hasMore),
	})
}

func (s *Server) handleApprovalRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/approvals/"), "/")
	if path == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if path == "batch" {
		s.handleBatchApprovalReview(w, r)
		return
	}
	parts := strings.Split(path, "/")
	approvalID := parts[0]
	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleApprovalDetail(w, r, approvalID)
		return
	}
	if len(parts) == 2 && parts[1] == "review" {
		s.handleApprovalReview(w, r, approvalID)
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

func (s *Server) handleBatchApprovalReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var input batchApprovalReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
		return
	}
	ids := dedupeTrimmed(input.ApprovalIDs)
	if len(ids) == 0 {
		writeErrorWithFields(w, http.StatusBadRequest, "invalid_batch", "At least one approval id is required", map[string]string{
			"approval_ids": "Provide one or more approval ids.",
		})
		return
	}

	ctx := r.Context()
	tenantID := currentTenant(r, s.cfg.DefaultTenant)
	updated := make([]domain.Approval, 0, len(ids))
	for _, approvalID := range ids {
		approval, err := s.repo.GetApproval(ctx, approvalID)
		if err != nil || approval.TenantID != tenantID {
			writeError(w, http.StatusNotFound, "approval_not_found", "Approval not found")
			return
		}
		approval, err = s.repo.ReviewApproval(ctx, approvalID, input.Approved, currentOperator(r), input.Note)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "approval_review_failed", err.Error())
			return
		}
		if approval.Kind == domain.ApprovalKindFlowStart && approval.Status == domain.ApprovalStatusApproved {
			rec := responseRecorder{header: make(http.Header)}
			s.dispatchFlow(&rec, r, approval.FlowID, approval.TenantID)
		}
		eventType, message := approvalEventInfo(approval.Status)
		s.recordAndFanoutEvent(ctx, approval.FlowID, eventType, message, map[string]any{
			"approval_id": approval.ID,
			"reviewed_by": approval.ReviewedBy,
			"status":      approval.Status,
		})
		updated = append(updated, approval)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"approvals": updated,
		"reviewed":  len(updated),
		"approved":  input.Approved,
	})
}

func (s *Server) handleApprovalDetail(w http.ResponseWriter, r *http.Request, approvalID string) {
	ctx := r.Context()
	approval, err := s.repo.GetApproval(ctx, approvalID)
	if err != nil || approval.TenantID != currentTenant(r, s.cfg.DefaultTenant) {
		writeError(w, http.StatusNotFound, "approval_not_found", "Approval not found")
		return
	}
	writeJSON(w, http.StatusOK, approval)
}

func (s *Server) handleApprovalReview(w http.ResponseWriter, r *http.Request, approvalID string) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()
	approval, err := s.repo.GetApproval(ctx, approvalID)
	if err != nil || approval.TenantID != currentTenant(r, s.cfg.DefaultTenant) {
		writeError(w, http.StatusNotFound, "approval_not_found", "Approval not found")
		return
	}
	var input domain.ApprovalReviewInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
		return
	}
	approval, err = s.repo.ReviewApproval(ctx, approvalID, input.Approved, currentOperator(r), input.Note)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "approval_review_failed", err.Error())
		return
	}
	if approval.Kind == domain.ApprovalKindFlowStart && approval.Status == domain.ApprovalStatusApproved {
		rec := responseRecorder{header: make(http.Header)}
		s.dispatchFlow(&rec, r, approval.FlowID, approval.TenantID)
	}
	eventType, message := approvalEventInfo(approval.Status)
	s.recordAndFanoutEvent(ctx, approval.FlowID, eventType, message, map[string]any{
		"approval_id": approval.ID,
		"reviewed_by": approval.ReviewedBy,
		"status":      approval.Status,
	})
	writeJSON(w, http.StatusOK, approval)
}
