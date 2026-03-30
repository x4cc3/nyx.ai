package reports

import (
	"strings"
	"testing"
	"time"

	"nyx/internal/domain"
)

func TestDocumentRendersMarkdownAndPDF(t *testing.T) {
	doc := Build(domain.Workspace{
		Flow: domain.Flow{
			ID:        "flow-1",
			Name:      "Acme Assessment",
			Target:    "https://app.example.com",
			Objective: "Validate report rendering",
			Status:    domain.StatusCompleted,
			CreatedAt: time.Now().UTC(),
		},
		TenantID:  "alpha",
		QueueMode: "jetstream",
		Findings: []domain.Finding{{
			Title:       "Export endpoint leak",
			Severity:    "high",
			Description: "Sensitive export was reachable.",
		}},
		Artifacts: []domain.Artifact{{
			Name:    "browser-snapshot",
			Kind:    "snapshot",
			Content: "/tmp/page.html",
		}},
	})

	markdown := doc.Markdown()
	if !strings.Contains(markdown, "# Acme Assessment Report") {
		t.Fatalf("expected markdown title, got %q", markdown)
	}
	if !strings.Contains(markdown, "Export endpoint leak") {
		t.Fatalf("expected finding in markdown, got %q", markdown)
	}

	pdf := doc.PDF()
	if len(pdf) < 5 || string(pdf[:5]) != "%PDF-" {
		t.Fatal("expected pdf output")
	}
}
