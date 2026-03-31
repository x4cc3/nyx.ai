package ids

import (
	"strings"
	"testing"
)

func TestNewID(t *testing.T) {
	id := New("browser")
	prefix := strings.SplitN(id, "_", 2)[0]
	if prefix != "browser" {
		t.Fatalf("expected prefix \"browser\", got %q", prefix)
	}
}

func TestNewUnique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := New("f")
		if seen[id] {
			t.Fatalf("duplicate ID: %s", id)
		}
		seen[id] = true
	}
}
