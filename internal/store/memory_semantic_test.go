package store

import (
	"context"
	"testing"
)

func TestMemoryStoreSemanticSearchAndRetentionPolicy(t *testing.T) {
	repo := NewMemoryStore()

	loginMemory, err := repo.AddMemory(context.Background(), "flow-1", "action-1", "observation", "Captured login page and session bootstrap details", map[string]string{
		"function_name": "browser",
	})
	if err != nil {
		t.Fatalf("add login memory: %v", err)
	}
	if loginMemory.Metadata["embedding_model"] == "" {
		t.Fatal("expected embedding metadata to be attached")
	}

	findingMemory, err := repo.AddMemory(context.Background(), "flow-1", "action-2", "finding", "Admin export endpoint accepted privileged parameters", nil)
	if err != nil {
		t.Fatalf("add finding memory: %v", err)
	}
	if findingMemory.Metadata["retention_policy"] != "long" {
		t.Fatalf("expected long retention for findings, got %s", findingMemory.Metadata["retention_policy"])
	}

	results, err := repo.SearchMemories(context.Background(), "flow-1", "login page bootstrap")
	if err != nil {
		t.Fatalf("search memories: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected semantic memory results")
	}
	if results[0].ID != loginMemory.ID {
		t.Fatalf("expected login memory to rank first, got %s", results[0].ID)
	}
}
