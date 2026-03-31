package domain

import "testing"

func TestNormalizeMemoryNamespace(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"all", ""},
		{"target_observations", MemoryNamespaceTargetObservations},
		{"exploit_references", MemoryNamespaceExploitReferences},
		{"reference_materials", MemoryNamespaceReferenceMaterials},
		{"operator_notes", MemoryNamespaceOperatorNotes},
		{"unknown", ""},
	}
	for _, tt := range tests {
		if got := NormalizeMemoryNamespace(tt.in); got != tt.want {
			t.Errorf("NormalizeMemoryNamespace(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestMemoryNamespace(t *testing.T) {
	if got := MemoryNamespace("exploit_reference", nil); got != MemoryNamespaceExploitReferences {
		t.Fatalf("expected exploit_references, got %q", got)
	}
	if got := MemoryNamespace("reference_material", nil); got != MemoryNamespaceReferenceMaterials {
		t.Fatalf("expected reference_materials, got %q", got)
	}
	if got := MemoryNamespace("operator_note", nil); got != MemoryNamespaceOperatorNotes {
		t.Fatalf("expected operator_notes, got %q", got)
	}
	if got := MemoryNamespace("unknown", nil); got != MemoryNamespaceTargetObservations {
		t.Fatalf("expected target_observations, got %q", got)
	}
}

func TestMemoryNamespaceFromMetadata(t *testing.T) {
	meta := map[string]string{"namespace": "exploit_references"}
	if got := MemoryNamespace("reference_material", meta); got != MemoryNamespaceExploitReferences {
		t.Fatalf("expected exploit_references, got %q", got)
	}
}
