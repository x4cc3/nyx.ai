package memory

import (
	"context"
	"strings"
	"testing"

	"nyx/internal/domain"
	"nyx/internal/store"
)

func TestSearchNamespaceFiltersMemoryClasses(t *testing.T) {
	repo := store.NewMemoryStore()
	service := New(repo)

	if _, err := service.StoreTargetObservation(context.Background(), "flow-1", "action-1", "Captured login form and CSRF bootstrap", nil); err != nil {
		t.Fatalf("store target observation: %v", err)
	}
	if _, err := service.StoreExploitReference(context.Background(), "flow-1", "action-2", "Exploit research: auth bypass PoC", nil); err != nil {
		t.Fatalf("store exploit reference: %v", err)
	}
	if _, err := service.StoreReferenceMaterial(context.Background(), "flow-1", "action-3", "Reference material: API authentication docs", nil); err != nil {
		t.Fatalf("store reference material: %v", err)
	}
	if _, err := service.StoreOperatorNote(context.Background(), "flow-1", "action-4", "Operator note: verify session reuse manually", nil); err != nil {
		t.Fatalf("store operator note: %v", err)
	}

	exploitResults := service.SearchNamespace(context.Background(), "flow-1", "auth bypass", domain.MemoryNamespaceExploitReferences)
	if len(exploitResults) != 1 {
		t.Fatalf("expected 1 exploit memory, got %d", len(exploitResults))
	}
	if exploitResults[0].Metadata["namespace"] != domain.MemoryNamespaceExploitReferences {
		t.Fatalf("expected exploit namespace, got %+v", exploitResults[0].Metadata)
	}

	operatorResults := service.SearchNamespace(context.Background(), "flow-1", "session reuse", domain.MemoryNamespaceOperatorNotes)
	if len(operatorResults) != 1 {
		t.Fatalf("expected 1 operator memory, got %d", len(operatorResults))
	}
	if operatorResults[0].Kind != "operator_note" {
		t.Fatalf("expected operator_note kind, got %q", operatorResults[0].Kind)
	}
}

func TestStoreActionResultCapturesExploitAndOperatorNoteSummaries(t *testing.T) {
	repo := store.NewMemoryStore()
	service := New(repo)

	stored := service.StoreActionResult(context.Background(), "flow-1", "action-1", "researcher", "search_exploits", map[string]string{
		"query": "app.example.com auth bypass exploit",
	}, map[string]string{
		"summary":        "Found auth bypass research and PoC references.",
		"source":         "sploitus",
		"result_1_title": "PoC A",
		"result_1_url":   "https://example.com/poc-a",
		"result_2_title": "Advisory B",
		"result_2_url":   "https://example.com/advisory-b",
	})
	if len(stored) != 1 {
		t.Fatalf("expected 1 stored exploit memory, got %d", len(stored))
	}
	if stored[0].Metadata["namespace"] != domain.MemoryNamespaceExploitReferences {
		t.Fatalf("expected exploit namespace, got %+v", stored[0].Metadata)
	}
	if !strings.Contains(stored[0].Content, "PoC A") || !strings.Contains(stored[0].Content, "sploitus") {
		t.Fatalf("expected summarized exploit references, got %q", stored[0].Content)
	}

	stored = service.StoreActionResult(context.Background(), "flow-1", "action-2", "operator", "file_write", map[string]string{
		"path":    "notes/mission-brief.txt",
		"content": "Target: https://app.example.com\nObjective: verify session reuse.\n",
	}, map[string]string{
		"summary": "Wrote workspace note.",
	})
	if len(stored) != 1 {
		t.Fatalf("expected 1 stored operator note, got %d", len(stored))
	}
	if stored[0].Metadata["namespace"] != domain.MemoryNamespaceOperatorNotes {
		t.Fatalf("expected operator namespace, got %+v", stored[0].Metadata)
	}
	if !strings.Contains(stored[0].Content, "notes/mission-brief.txt") {
		t.Fatalf("expected operator note path in content, got %q", stored[0].Content)
	}
}
