package executor

import "testing"

func TestResolveToolsetHonorsExplicitSelection(t *testing.T) {
	if got := resolveToolset(map[string]string{"toolset": "pentest"}); got != toolsetPentest {
		t.Fatalf("expected explicit pentest toolset, got %s", got)
	}
	if got := resolveToolset(map[string]string{"toolset": "general"}); got != toolsetGeneral {
		t.Fatalf("expected explicit general toolset, got %s", got)
	}
}

func TestResolveToolsetDefaultsToPentestForSecurityWork(t *testing.T) {
	got := resolveToolset(map[string]string{
		"goal": "run controlled reconnaissance and vulnerability validation",
	})
	if got != toolsetPentest {
		t.Fatalf("expected pentest toolset, got %s", got)
	}
}

func TestResolveToolsetDefaultsToGeneralOtherwise(t *testing.T) {
	got := resolveToolset(map[string]string{
		"goal": "write a workspace note for the operator",
	})
	if got != toolsetGeneral {
		t.Fatalf("expected general toolset, got %s", got)
	}
}
