package agentruntime

import (
	"fmt"
	"sort"
	"strings"

	"nyx/internal/domain"
)

type PromptTemplate struct {
	Role         string
	Model        string
	SystemPrompt string
	Focus        []string
}

type PromptContext struct {
	Flow               domain.Flow
	TaskName           string
	SubtaskName        string
	MemorySummary      string
	AvailableFunctions []domain.FunctionDef
}

type PromptLibrary struct {
	templates map[string]PromptTemplate
	fallback  PromptTemplate
}

func DefaultPromptLibrary(defaultModel string) PromptLibrary {
	if strings.TrimSpace(defaultModel) == "" {
		defaultModel = "gpt-5.1-codex-mini"
	}
	return PromptLibrary{
		templates: map[string]PromptTemplate{
			"planner": {
				Role:         "planner",
				Model:        defaultModel,
				SystemPrompt: "Decompose the engagement into practical tasks, watch for blockers, and hand off clean execution context to specialists.",
				Focus: []string{
					"keep the plan narrow and reviewable",
					"identify the next highest-signal action",
					"escalate quickly when repeated failures appear",
				},
			},
			"researcher": {
				Role:         "researcher",
				Model:        defaultModel,
				SystemPrompt: "Collect target context and observations that make later execution safer and more accurate. Prefer browser and search evidence before terminal-heavy validation.",
				Focus: []string{
					"validate target shape before deeper execution",
					"capture durable observations in memory",
					"hand off evidence the executor can immediately use",
				},
			},
			"executor": {
				Role:         "executor",
				Model:        defaultModel,
				SystemPrompt: "Operate like a NYX pentest executor. Run the smallest controlled validation step that advances the subtask, treat terminal execution as a security toolbox rather than a generic shell, and preserve operator-visible evidence after each meaningful step.",
				Focus: []string{
					"prefer browser and search reconnaissance before terminal validation when possible",
					"stay inside scoped execution profiles and approved network settings",
					"emit structured artifacts and workspace notes",
					"stop and escalate after repeated failure or unclear scope",
				},
			},
			"operator": {
				Role:         "operator",
				Model:        defaultModel,
				SystemPrompt: "Request operator input when runtime supervision determines the flow is blocked or risky.",
				Focus: []string{
					"explain the blocker clearly",
					"include the failed subtask and next safe option",
				},
			},
		},
		fallback: PromptTemplate{
			Role:         "specialist",
			Model:        defaultModel,
			SystemPrompt: "Complete the assigned NYX subtask with concise, observable output.",
			Focus: []string{
				"prefer minimal controlled actions",
			},
		},
	}
}

func (l PromptLibrary) Template(role string) PromptTemplate {
	key := strings.ToLower(strings.TrimSpace(role))
	if tpl, ok := l.templates[key]; ok {
		return tpl
	}
	return l.fallback
}

func (l PromptLibrary) Render(role string, ctx PromptContext) string {
	tpl := l.Template(role)

	var b strings.Builder
	b.WriteString(tpl.SystemPrompt)
	b.WriteString("\n\nMission:\n")
	fmt.Fprintf(&b, "- Flow: %s\n", ctx.Flow.Name)
	fmt.Fprintf(&b, "- Target: %s\n", ctx.Flow.Target)
	fmt.Fprintf(&b, "- Objective: %s\n", ctx.Flow.Objective)
	fmt.Fprintf(&b, "- Task: %s\n", ctx.TaskName)
	fmt.Fprintf(&b, "- Subtask: %s\n", ctx.SubtaskName)
	if ctx.MemorySummary != "" {
		fmt.Fprintf(&b, "- Prior memory: %s\n", ctx.MemorySummary)
	}
	if len(tpl.Focus) > 0 {
		b.WriteString("\nFocus:\n")
		for _, item := range tpl.Focus {
			b.WriteString("- ")
			b.WriteString(item)
			b.WriteByte('\n')
		}
	}
	if catalog := renderFunctionCatalog(ctx.AvailableFunctions); catalog != "" {
		b.WriteString("\nAvailable Functions:\n")
		b.WriteString(catalog)
	}
	if strings.EqualFold(strings.TrimSpace(role), "executor") {
		if guidance := renderExecutorGuidance(ctx.AvailableFunctions); guidance != "" {
			b.WriteString("\nExecution Guidance:\n")
			b.WriteString(guidance)
		}
	}
	return strings.TrimSpace(b.String())
}

func renderFunctionCatalog(defs []domain.FunctionDef) string {
	if len(defs) == 0 {
		return ""
	}
	order := []string{"environment", "search_network", "search_vector_db", "barrier"}
	labels := map[string]string{
		"environment":      "environment",
		"search_network":   "browser and network research",
		"search_vector_db": "memory",
		"barrier":          "control",
	}
	grouped := make(map[string][]string, len(order))
	extra := make([]string, 0)
	for _, def := range defs {
		category := strings.TrimSpace(def.Category)
		if category == "" {
			category = strings.TrimSpace(def.Profile)
		}
		if category == "" {
			category = "other"
		}
		grouped[category] = append(grouped[category], def.Name)
		if _, ok := labels[category]; !ok && category != "other" {
			labels[category] = category
			extra = append(extra, category)
		}
	}
	sort.Strings(extra)
	order = append(order, extra...)
	if len(grouped["other"]) > 0 {
		order = append(order, "other")
		labels["other"] = "other"
	}
	var b strings.Builder
	seen := make(map[string]struct{}, len(order))
	for _, category := range order {
		if _, ok := seen[category]; ok {
			continue
		}
		seen[category] = struct{}{}
		names := grouped[category]
		if len(names) == 0 {
			continue
		}
		sort.Strings(names)
		fmt.Fprintf(&b, "- %s: %s\n", labels[category], strings.Join(names, ", "))
	}
	return b.String()
}

func renderExecutorGuidance(defs []domain.FunctionDef) string {
	hasPentestPath := false
	for _, def := range defs {
		if def.RequiresPentestImage || def.Name == "terminal_exec" {
			hasPentestPath = true
			break
		}
	}
	var b strings.Builder
	if hasPentestPath {
		b.WriteString("- Pentest image path is available. Assume the pentest worker image contains: curl, jq, subfinder, dnsx, assetfinder, httpx, katana, gau, waybackurls, ffuf, gobuster, nuclei, naabu, nmap, nikto, whatweb, sqlmap, python3.\n")
		b.WriteString("- Use `terminal_exec` for explicit pentest-capable commands. Use `terminal` with `toolset=pentest` when the plan already carries that context.\n")
	} else {
		b.WriteString("- No pentest-image-backed terminal path is exposed. Verify tool availability with a minimal version/help command before depending on any binary.\n")
	}
	b.WriteString("- Recon guidance: start with browser_html, browser_markdown, browser_links, search_web, search_deep, or search_code. Move to terminal recon only when network policy allows and the next hypothesis is concrete.\n")
	b.WriteString("- Web testing guidance: prefer httpx, ffuf, nuclei, whatweb, nikto, sqlmap, gobuster, curl, and jq for targeted validation tied to a specific path, parameter, header, or service.\n")
	b.WriteString("- Exploit validation guidance: search for advisories, PoCs, and version clues first. Then run the smallest proof step needed to confirm or kill the exploit hypothesis.\n")
	b.WriteString("- Evidence preservation: keep screenshots, HTML snapshots, stdout, stderr, notes, and artifact paths operator-visible after each meaningful action.\n")
	b.WriteString("- Safe escalation: ask for help when scope is unclear, credentials or session material are missing, the action would be high-risk, or repeated attempts are failing without producing new evidence.\n")
	return b.String()
}
