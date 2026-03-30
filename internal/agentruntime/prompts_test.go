package agentruntime

import (
	"strings"
	"testing"

	"nyx/internal/domain"
)

func TestRenderExecutorPromptIncludesPentestToolCatalogAndGuidance(t *testing.T) {
	library := DefaultPromptLibrary("gpt-5.1-codex-mini")

	prompt := library.Render("executor", PromptContext{
		Flow: domain.Flow{
			Name:      "Acme external assessment",
			Target:    "https://app.example.com",
			Objective: "Validate the login and admin surface safely.",
		},
		TaskName:      "Execution",
		SubtaskName:   "Controlled validation",
		MemorySummary: "Login page discovered; admin path exposed in robots.txt.",
		AvailableFunctions: []domain.FunctionDef{
			{Name: "terminal_exec", Profile: "terminal", Category: "environment", RequiresPentestImage: true},
			{Name: "browser_html", Profile: "browser", Category: "search_network", RequiresNetwork: true},
			{Name: "browser_screenshot", Profile: "browser", Category: "search_network", RequiresNetwork: true},
			{Name: "search_deep", Profile: "search", Category: "search_network", RequiresNetwork: true},
			{Name: "search_exploits", Profile: "search", Category: "search_network", RequiresNetwork: true},
			{Name: "search_code", Profile: "search", Category: "search_network", RequiresNetwork: true},
			{Name: "search_memory", Profile: "memory", Category: "search_vector_db"},
			{Name: "ask", Profile: "control", Category: "barrier"},
		},
	})

	for _, want := range []string{
		"Available Functions:",
		"- environment: terminal_exec",
		"- browser and network research: browser_html, browser_screenshot, search_code, search_deep, search_exploits",
		"Pentest image path is available.",
		"subfinder, dnsx, assetfinder, httpx, katana",
		"Use `terminal_exec` for explicit pentest-capable commands.",
		"Recon guidance:",
		"Web testing guidance:",
		"Exploit validation guidance:",
		"Evidence preservation:",
		"Safe escalation:",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("expected prompt to contain %q\nprompt:\n%s", want, prompt)
		}
	}
}

func TestRenderExecutorPromptFallsBackToVerifyToolsWhenPentestImageUnavailable(t *testing.T) {
	library := DefaultPromptLibrary("gpt-5.1-codex-mini")

	prompt := library.Render("executor", PromptContext{
		Flow: domain.Flow{
			Name:      "Acme assessment",
			Target:    "https://app.example.com",
			Objective: "Inspect public pages.",
		},
		TaskName:    "Execution",
		SubtaskName: "Browser-first triage",
		AvailableFunctions: []domain.FunctionDef{
			{Name: "terminal", Profile: "terminal", Category: "environment"},
			{Name: "browser_markdown", Profile: "browser", Category: "search_network", RequiresNetwork: true},
			{Name: "search_web", Profile: "search", Category: "search_network", RequiresNetwork: true},
		},
	})

	if !strings.Contains(prompt, "Verify tool availability with a minimal version/help command before depending on any binary.") {
		t.Fatalf("expected verify-tools guidance, got:\n%s", prompt)
	}
	if strings.Contains(prompt, "Pentest image path is available.") {
		t.Fatalf("did not expect pentest-image guidance, got:\n%s", prompt)
	}
}
