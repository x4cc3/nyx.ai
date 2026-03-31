package ids

import (
	"strings"
	"testing"
)

func TestNewPrefix(t *testing.T) {
	id := New("flow")
	parts := strings.SplitN(id, "_", 2)
	if parts[0] != "flow" {
		t.Fatalf("expected prefix \"flow\", got %q", parts[0])
	}
	if len(parts[1]) != 12 {
		t.Fatalf("expected 12 hex chars suffix, got %d chars", len(parts[1]))
	}
}

func TestNewIsUnique(t *testing.T) {
	id1 := New("task")
	id2 := New("task")
	if id1 == id2 {
		t.Fatal("expected unique IDs, got duplicate")
	}
}

func TestNewLowercasePrefix(t *testing.T) {
	id := New("FLOW")
	if !strings.HasPrefix(id, "flow_") {
		t.Fatalf("expected lowercase prefix, got %q", id)
	}
}
