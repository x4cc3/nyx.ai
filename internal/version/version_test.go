package version

import "testing"

func TestStringDefault(t *testing.T) {
	s := String()
	if s != "dev (unknown, unknown)" {
		t.Fatalf("expected \"dev (unknown, unknown)\", got %q", s)
	}
}

func TestStringWithValues(t *testing.T) {
	Version, Commit, BuildDate = "v1.0.0", "abc123", "2026-03-31"
	defer func() { Version, Commit, BuildDate = "dev", "unknown", "unknown" }()

	s := String()
	if s != "v1.0.0 (abc123, 2026-03-31)" {
		t.Fatalf("expected formatted version, got %q", s)
	}
}
