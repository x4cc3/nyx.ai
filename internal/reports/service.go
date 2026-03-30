package reports

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"nyx/internal/domain"
)

type Document struct {
	Title     string           `json:"title"`
	Generated time.Time        `json:"generated_at"`
	Summary   map[string]any   `json:"summary"`
	Workspace domain.Workspace `json:"workspace"`
}

func Build(workspace domain.Workspace) Document {
	return Document{
		Title:     workspace.Flow.Name + " Report",
		Generated: time.Now().UTC(),
		Summary: map[string]any{
			"status":          workspace.Flow.Status,
			"findings":        len(workspace.Findings),
			"artifacts":       len(workspace.Artifacts),
			"memories":        len(workspace.Memories),
			"approvals":       len(workspace.Approvals),
			"pending_review":  workspace.NeedsReview,
			"tenant_id":       workspace.TenantID,
			"queue_mode":      workspace.QueueMode,
			"functions_count": len(workspace.Functions),
		},
		Workspace: workspace,
	}
}

func (d Document) JSON() ([]byte, error) {
	return json.MarshalIndent(d, "", "  ")
}

func (d Document) Markdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", d.Title)
	fmt.Fprintf(&b, "- Generated: %s\n", d.Generated.Format(time.RFC3339))
	fmt.Fprintf(&b, "- Tenant: %s\n", d.Workspace.TenantID)
	fmt.Fprintf(&b, "- Queue mode: %s\n", d.Workspace.QueueMode)
	fmt.Fprintf(&b, "- Status: %s\n", d.Workspace.Flow.Status)
	fmt.Fprintf(&b, "- Target: %s\n", d.Workspace.Flow.Target)
	fmt.Fprintf(&b, "- Objective: %s\n\n", d.Workspace.Flow.Objective)

	writeSection := func(title string) {
		fmt.Fprintf(&b, "## %s\n\n", title)
	}

	writeSection("Findings")
	if len(d.Workspace.Findings) == 0 {
		b.WriteString("No findings recorded.\n\n")
	} else {
		for _, finding := range d.Workspace.Findings {
			fmt.Fprintf(&b, "- **%s** (%s): %s\n", finding.Title, finding.Severity, finding.Description)
		}
		b.WriteString("\n")
	}

	writeSection("Artifacts")
	if len(d.Workspace.Artifacts) == 0 {
		b.WriteString("No artifacts recorded.\n\n")
	} else {
		for _, artifact := range d.Workspace.Artifacts {
			fmt.Fprintf(&b, "- `%s` [%s]: %s\n", artifact.Name, artifact.Kind, artifact.Content)
		}
		b.WriteString("\n")
	}

	writeSection("Memories")
	if len(d.Workspace.Memories) == 0 {
		b.WriteString("No memory entries recorded.\n\n")
	} else {
		for _, memory := range d.Workspace.Memories {
			fmt.Fprintf(&b, "- [%s] %s\n", memory.Kind, memory.Content)
		}
		b.WriteString("\n")
	}

	writeSection("Approvals")
	if len(d.Workspace.Approvals) == 0 {
		b.WriteString("No approvals recorded.\n\n")
	} else {
		for _, approval := range d.Workspace.Approvals {
			fmt.Fprintf(&b, "- **%s** %s by %s", approval.Status, approval.Kind, approval.RequestedBy)
			if approval.ReviewedBy != "" {
				fmt.Fprintf(&b, " reviewed by %s", approval.ReviewedBy)
			}
			if approval.ReviewNote != "" {
				fmt.Fprintf(&b, " (%s)", approval.ReviewNote)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	writeSection("Actions")
	if len(d.Workspace.Actions) == 0 {
		b.WriteString("No actions recorded.\n")
	} else {
		for _, action := range d.Workspace.Actions {
			fmt.Fprintf(&b, "- `%s` via %s (%s)\n", action.FunctionName, action.AgentRole, action.Status)
		}
	}

	return b.String()
}

func (d Document) PDF() []byte {
	text := d.Markdown()
	lines := strings.Split(text, "\n")
	if len(lines) > 48 {
		lines = lines[:48]
	}
	content := "BT /F1 11 Tf 50 780 Td 14 TL\n"
	for i, line := range lines {
		if i > 0 {
			content += "T*\n"
		}
		content += fmt.Sprintf("(%s) Tj\n", escapePDFText(line))
	}
	content += "ET"

	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content),
	}

	var b bytes.Buffer
	offsets := make([]int, len(objects)+1)
	b.WriteString("%PDF-1.4\n")
	for i, object := range objects {
		offsets[i+1] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, object)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n", len(objects)+1)
	b.WriteString("0000000000 65535 f \n")
	for i := 1; i < len(offsets); i++ {
		fmt.Fprintf(&b, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&b, "trailer << /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF", len(objects)+1, xref)
	return b.Bytes()
}

func escapePDFText(input string) string {
	input = strings.ReplaceAll(input, `\`, `\\`)
	input = strings.ReplaceAll(input, "(", `\(`)
	input = strings.ReplaceAll(input, ")", `\)`)
	return input
}
