package agentruntime

import (
	"context"
	"strings"

	"nyx/internal/domain"
)

type ActionPolicyMetadata struct {
	Model            string
	ResponseID       string
	ReasoningSummary string
}

type ActionObservation struct {
	FunctionName string
	Input        map[string]string
	Output       map[string]string
	Error        string
}

type ActionPolicyRequest struct {
	Flow               domain.Flow
	Task               TaskSpec
	Subtask            SubtaskSpec
	Attempt            int
	Turn               int
	MemorySummary      string
	AvailableFunctions []domain.FunctionDef
	PlannedSteps       []ActionSpec
	Observations       []ActionObservation
}

type ActionDecision struct {
	FunctionName string            `json:"function_name"`
	Input        map[string]string `json:"input"`
}

type ActionPolicy interface {
	NextAction(context.Context, ActionPolicyRequest) (ActionDecision, ActionPolicyMetadata, error)
}

func WithActionPolicy(policy ActionPolicy) Option {
	return func(runtime *Runtime) {
		runtime.actionPolicy = policy
	}
}

func sanitizeActionDecision(decision ActionDecision, defs []domain.FunctionDef, fallback ActionSpec) ActionSpec {
	allowed := make(map[string]struct{}, len(defs))
	for _, def := range defs {
		allowed[def.Name] = struct{}{}
	}
	name := strings.TrimSpace(decision.FunctionName)
	if _, ok := allowed[name]; !ok {
		return fallback
	}
	return ActionSpec{
		FunctionName: name,
		Input:        trimStringMap(decision.Input, 32, 4096),
	}
}
