package orchestrator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"nyx/internal/domain"
	"nyx/internal/functions"
)

func (o *Orchestrator) maybeRequireRiskApproval(ctx context.Context, flowID, actionID, functionName string, input map[string]string) (functions.CallResult, bool) {
	if !o.requireRiskyApproval {
		return functions.CallResult{}, false
	}
	assessment := functions.AssessTerminalRisk(functionName, input)
	if !assessment.RequiresApproval() {
		return functions.CallResult{}, false
	}

	approval, approved, err := o.ensureRiskApproval(ctx, flowID, actionID, functionName, input, assessment)
	if err != nil {
		return functions.CallResult{
			Profile: "control",
			Runtime: "go-orchestrator",
			Output: map[string]string{
				"summary": "Failed to create operator approval for high-risk terminal execution.",
				"error":   err.Error(),
			},
			Err: err,
		}, true
	}
	if approved {
		return functions.CallResult{}, false
	}

	summary := fmt.Sprintf("Operator approval required before running a high-risk terminal action (%s). Approve the request and restart the flow to continue.", assessment.Summary())
	output := functions.ApprovalRequiredOutput(summary, assessment)
	output["approval_id"] = approval.ID
	output["approval_kind"] = approval.Kind
	output["requested_by"] = approval.RequestedBy
	return functions.CallResult{
		Profile: "control",
		Runtime: "go-orchestrator",
		Output:  output,
		Err:     functions.ErrApprovalRequired,
	}, true
}

func (o *Orchestrator) ensureRiskApproval(ctx context.Context, flowID, actionID, functionName string, input map[string]string, assessment functions.TerminalRiskAssessment) (domain.Approval, bool, error) {
	approvals, err := o.repo.ListApprovalsByFlow(ctx, flowID)
	if err != nil {
		return domain.Approval{}, false, err
	}
	key := riskApprovalKey(functionName, input, assessment)
	for _, approval := range approvals {
		if approval.Kind != domain.ApprovalKindRiskReview {
			continue
		}
		if approval.Payload["risk_key"] != key {
			continue
		}
		if approval.Status == domain.ApprovalStatusApproved {
			return approval, true, nil
		}
		if approval.Status == domain.ApprovalStatusPending {
			return approval, false, nil
		}
	}

	approval, err := o.repo.CreateApproval(ctx, flowID, "", domain.ApprovalKindRiskReview, "nyx-orchestrator", riskApprovalReason(assessment), map[string]string{
		"action_id":        actionID,
		"function_name":    functionName,
		"target":           strings.TrimSpace(input["target"]),
		"command":          strings.TrimSpace(input["command"]),
		"toolset":          strings.TrimSpace(input["toolset"]),
		"risk_key":         key,
		"risk_categories":  assessment.CategoryList(),
		"network_required": strings.TrimSpace(input["network_required"]),
	})
	if err != nil {
		return domain.Approval{}, false, err
	}
	_, _ = o.repo.RecordEvent(ctx, flowID, domain.EventActionApprovalRequested, "Action approval requested", map[string]any{
		"action_id":       actionID,
		"approval_id":     approval.ID,
		"function_name":   functionName,
		"risk_categories": assessment.Categories,
	})
	return approval, false, nil
}

func riskApprovalReason(assessment functions.TerminalRiskAssessment) string {
	return fmt.Sprintf("High-risk terminal action requires operator approval before execution: %s.", assessment.Summary())
}

func riskApprovalKey(functionName string, input map[string]string, assessment functions.TerminalRiskAssessment) string {
	payload := strings.Join([]string{
		strings.TrimSpace(functionName),
		strings.TrimSpace(input["target"]),
		strings.TrimSpace(input["command"]),
		strings.TrimSpace(input["toolset"]),
		assessment.CategoryList(),
	}, "\n")
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}
