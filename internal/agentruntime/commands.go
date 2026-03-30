package agentruntime

import (
	"fmt"
	"net/url"
	"strings"

	"nyx/internal/domain"
)

func (r *Runtime) decompose(flow domain.Flow) FlowPlan {
	target := strings.TrimSpace(flow.Target)
	objective := strings.TrimSpace(flow.Objective)
	if objective == "" {
		objective = "Validate the target safely with controlled runtime actions."
	}
	researchSteps := r.buildResearchSteps(target, objective)
	exploitSteps := r.buildExploitResearchSteps(target, objective)
	baselineValidation := r.buildBaselineValidationSteps(target, objective)
	targetedValidation := r.buildTargetedValidationSteps(target, objective)
	evidenceNote := r.buildEvidenceNoteStep(target, objective)

	researchSubtasks := []SubtaskSpec{
		{
			Role:        "researcher",
			Name:        "Surface profiling",
			Description: "Inspect the target and capture durable observations for later specialists.",
			MaxAttempts: 2,
			EscalateTo:  "planner",
			Steps:       researchSteps,
		},
	}
	if len(exploitSteps) > 0 {
		researchSubtasks = append(researchSubtasks, SubtaskSpec{
			Role:        "researcher",
			Name:        "Exploit reference triage",
			Description: "Collect exploit references, advisories, and code clues before active validation.",
			MaxAttempts: 2,
			EscalateTo:  "planner",
			Steps:       exploitSteps,
		})
	}

	executionSubtasks := []SubtaskSpec{
		{
			Role:         "executor",
			Name:         "Baseline target fingerprint",
			Description:  "Run a concrete low-noise fingerprinting step inside the pentest toolchain.",
			MaxAttempts:  2,
			EscalateTo:   "planner",
			EscalateText: "The executor exhausted the baseline fingerprinting attempts and needs operator guidance.",
			Steps:        baselineValidation,
		},
	}
	if len(targetedValidation) > 0 {
		executionSubtasks = append(executionSubtasks, SubtaskSpec{
			Role:         "executor",
			Name:         "Hypothesis-driven validation",
			Description:  "Use targeted security tools that match the current objective and preserve evidence from the result.",
			MaxAttempts:  2,
			EscalateTo:   "planner",
			EscalateText: "The executor exhausted the targeted validation attempts and needs operator guidance.",
			Steps:        targetedValidation,
		})
	}
	executionSubtasks = append(executionSubtasks, SubtaskSpec{
		Role:        "executor",
		Name:        "Workspace evidence note",
		Description: "Write a mission note into the isolated workspace so the operator can inspect runtime artifacts.",
		MaxAttempts: 1,
		EscalateTo:  "planner",
		Steps:       []ActionSpec{evidenceNote},
	})

	return FlowPlan{
		Tasks: []TaskSpec{
			{
				Role:        "researcher",
				Name:        "Target research",
				Description: "Collect target context, scope clues, and likely attack surface for downstream execution.",
				Subtasks:    researchSubtasks,
			},
			{
				Role:        "executor",
				Name:        "Execution",
				Description: "Run controlled validation steps with concrete pentest tooling and preserve evidence inside isolated execution profiles.",
				Subtasks:    executionSubtasks,
			},
		},
	}
}

func (r *Runtime) enrichPlan(flow domain.Flow, plan FlowPlan) FlowPlan {
	out := FlowPlan{Tasks: make([]TaskSpec, 0, len(plan.Tasks))}
	for _, task := range plan.Tasks {
		nextTask := task
		nextTask.Subtasks = make([]SubtaskSpec, 0, len(task.Subtasks))
		for _, subtask := range task.Subtasks {
			nextSubtask := subtask
			nextSubtask.Steps = make([]ActionSpec, 0, len(subtask.Steps))
			for _, step := range subtask.Steps {
				nextSubtask.Steps = append(nextSubtask.Steps, r.enrichActionSpec(flow, task, subtask, step))
			}
			nextTask.Subtasks = append(nextTask.Subtasks, nextSubtask)
		}
		out.Tasks = append(out.Tasks, nextTask)
	}
	return out
}

func (r *Runtime) enrichActionSpec(flow domain.Flow, task TaskSpec, subtask SubtaskSpec, step ActionSpec) ActionSpec {
	input := cloneStringMap(step.Input)
	if input == nil {
		input = map[string]string{}
	}
	target := strings.TrimSpace(flow.Target)
	objective := strings.TrimSpace(flow.Objective)
	if input["target"] == "" {
		input["target"] = target
	}
	switch step.FunctionName {
	case "browser":
		if input["url"] == "" && isWebTarget(target) {
			input["url"] = target
		}
	case "browser_markdown":
		defaultActionMetadata(input, "false", "page_markdown", "markdown")
		if input["url"] == "" && isWebTarget(target) {
			input["url"] = target
		}
	case "browser_html":
		defaultActionMetadata(input, "false", "page_html", "html")
		if input["url"] == "" && isWebTarget(target) {
			input["url"] = target
		}
	case "browser_links":
		defaultActionMetadata(input, "false", "link_inventory", "links")
		if input["url"] == "" && isWebTarget(target) {
			input["url"] = target
		}
	case "browser_screenshot":
		defaultActionMetadata(input, "false", "screenshot_path", "screenshot_path")
		if input["url"] == "" && isWebTarget(target) {
			input["url"] = target
		}
	case "search_web":
		defaultActionMetadata(input, "false", "web_search_summary", "summary")
		if input["target"] == "" {
			input["target"] = target
		}
	case "search_deep":
		defaultActionMetadata(input, "false", "deep_research_summary", "summary")
	case "search_code":
		defaultActionMetadata(input, "false", "code_reference", "summary")
	case "search_exploits":
		defaultActionMetadata(input, "false", "exploit_reference", "summary")
	case "search_memory":
		defaultActionMetadata(input, "false", "memory_hits", "summary")
		if input["namespace"] == "" {
			input["namespace"] = domain.MemoryNamespaceTargetObservations
		}
	case "terminal", "terminal_exec":
		defaultActionMetadata(input, "true", "terminal_output", "stdout")
		if input["toolset"] == "" {
			input["toolset"] = "pentest"
		}
		if strings.TrimSpace(input["goal"]) == "" {
			input["goal"] = fmt.Sprintf("%s / %s / %s", task.Name, subtask.Name, objective)
		}
		if strings.TrimSpace(input["command"]) == "" {
			if command := r.deriveTerminalCommand(target, objective, task, subtask); command != "" {
				input["command"] = command
				if r.hasFunction("terminal_exec") {
					step.FunctionName = "terminal_exec"
				}
			}
		}
	case "file":
		if input["operation"] == "write" || strings.TrimSpace(input["content"]) != "" {
			defaultActionMetadata(input, "false", "workspace_note", "file_path")
		}
	case "file_write":
		defaultActionMetadata(input, "false", "workspace_note", "file_path")
	}
	step.Input = input
	return step
}

func defaultActionMetadata(input map[string]string, networkRequired, evidenceExpected, preferredOutput string) {
	if input["network_required"] == "" {
		input["network_required"] = networkRequired
	}
	if input["evidence_expected"] == "" {
		input["evidence_expected"] = evidenceExpected
	}
	if input["preferred_output"] == "" {
		input["preferred_output"] = preferredOutput
	}
}

func (r *Runtime) buildResearchSteps(target, objective string) []ActionSpec {
	steps := make([]ActionSpec, 0, 5)
	if isWebTarget(target) {
		if r.hasFunction("browser_screenshot") {
			steps = append(steps, ActionSpec{FunctionName: "browser_screenshot", Input: map[string]string{"url": target, "wait_selector": "body", "preferred_output": "screenshot_path", "evidence_expected": "screenshot_path"}})
		}
		if r.hasFunction("browser_markdown") {
			steps = append(steps, ActionSpec{FunctionName: "browser_markdown", Input: map[string]string{"url": target}})
		} else if r.hasFunction("browser_html") {
			steps = append(steps, ActionSpec{FunctionName: "browser_html", Input: map[string]string{"url": target}})
		} else if r.hasFunction("browser") {
			steps = append(steps, ActionSpec{FunctionName: "browser", Input: map[string]string{"url": target}})
		}
		if r.hasFunction("browser_links") {
			steps = append(steps, ActionSpec{FunctionName: "browser_links", Input: map[string]string{"url": target}})
		}
	}
	if r.hasFunction("search_deep") {
		steps = append(steps, ActionSpec{FunctionName: "search_deep", Input: map[string]string{"query": composeResearchQuery(target, objective)}})
	} else if r.hasFunction("search_web") {
		steps = append(steps, ActionSpec{FunctionName: "search_web", Input: map[string]string{"target": target, "objective": objective}})
	}
	if r.hasFunction("search_code") {
		steps = append(steps, ActionSpec{FunctionName: "search_code", Input: map[string]string{"query": composeCodeQuery(target, objective)}})
	}
	if r.hasFunction("search_memory") {
		steps = append(steps, ActionSpec{FunctionName: "search_memory", Input: map[string]string{"query": "target snapshot " + target, "namespace": domain.MemoryNamespaceTargetObservations}})
	}
	if len(steps) == 0 {
		return []ActionSpec{{FunctionName: "search_memory", Input: map[string]string{"query": "subtask context", "namespace": domain.MemoryNamespaceTargetObservations}}}
	}
	return steps
}

func (r *Runtime) buildExploitResearchSteps(target, objective string) []ActionSpec {
	if !r.hasFunction("search_exploits") {
		return nil
	}
	steps := []ActionSpec{{
		FunctionName: "search_exploits",
		Input: map[string]string{
			"query":             composeExploitQuery(target, objective),
			"preferred_output":  "summary",
			"evidence_expected": "exploit_reference",
		},
	}}
	if r.hasFunction("search_memory") {
		steps = append(steps, ActionSpec{
			FunctionName: "search_memory",
			Input: map[string]string{
				"query":     composeExploitQuery(target, objective),
				"namespace": domain.MemoryNamespaceExploitReferences,
			},
		})
	}
	return steps
}

func (r *Runtime) buildBaselineValidationSteps(target, objective string) []ActionSpec {
	functionName := r.preferredTerminalFunction()
	if functionName == "" {
		return []ActionSpec{{FunctionName: "search_memory", Input: map[string]string{"query": "execution fallback context", "namespace": domain.MemoryNamespaceTargetObservations}}}
	}
	command := buildBaselineCommand(target)
	input := map[string]string{
		"goal":              fmt.Sprintf("capture a low-noise baseline fingerprint for %s while pursuing %q", target, objective),
		"command":           command,
		"toolset":           "pentest",
		"network_required":  "true",
		"evidence_expected": "terminal_baseline",
		"preferred_output":  "stdout",
	}
	return []ActionSpec{{FunctionName: functionName, Input: input}}
}

func (r *Runtime) buildTargetedValidationSteps(target, objective string) []ActionSpec {
	functionName := r.preferredTerminalFunction()
	if functionName == "" {
		return nil
	}
	steps := make([]ActionSpec, 0, 3)
	if shouldUseContentDiscovery(objective) {
		steps = append(steps, ActionSpec{
			FunctionName: functionName,
			Input: map[string]string{
				"goal":              fmt.Sprintf("check common discovery paths on %s that may support the objective %q", target, objective),
				"command":           buildFFUFCommand(target),
				"toolset":           "pentest",
				"network_required":  "true",
				"evidence_expected": "content_discovery",
				"preferred_output":  "stdout",
			},
		})
	}
	if shouldUseTemplateScan(objective) {
		steps = append(steps, ActionSpec{
			FunctionName: functionName,
			Input: map[string]string{
				"goal":              fmt.Sprintf("run a narrow template-based check against %s for objective %q", target, objective),
				"command":           buildNucleiCommand(target),
				"toolset":           "pentest",
				"network_required":  "true",
				"evidence_expected": "nuclei_findings",
				"preferred_output":  "stdout",
			},
		})
	}
	if shouldUseSQLValidation(target, objective) {
		steps = append(steps, ActionSpec{
			FunctionName: functionName,
			Input: map[string]string{
				"goal":              fmt.Sprintf("run a low-risk sqlmap check aligned to objective %q", objective),
				"command":           buildSQLMapCommand(target),
				"toolset":           "pentest",
				"network_required":  "true",
				"evidence_expected": "sqlmap_output",
				"preferred_output":  "stdout",
			},
		})
	}
	return steps
}

func (r *Runtime) buildEvidenceNoteStep(target, objective string) ActionSpec {
	functionName := "file"
	input := map[string]string{
		"path":              "notes/mission-brief.txt",
		"content":           fmt.Sprintf("Target: %s\nObjective: %s\nRuntime: supervised agent loop\nEvidence: preserve screenshots, HTML snapshots, stdout, stderr, and search summaries.\n", target, objective),
		"preferred_output":  "file_path",
		"evidence_expected": "workspace_note",
	}
	if r.hasFunction("file_write") {
		functionName = "file_write"
	} else {
		input["operation"] = "write"
	}
	return ActionSpec{FunctionName: functionName, Input: input}
}

func (r *Runtime) preferredTerminalFunction() string {
	if r.hasFunction("terminal_exec") {
		return "terminal_exec"
	}
	if r.hasFunction("terminal") {
		return "terminal"
	}
	return ""
}

func (r *Runtime) hasFunction(name string) bool {
	for _, def := range r.defs {
		if def.Name == name {
			return true
		}
	}
	return false
}

func (r *Runtime) deriveTerminalCommand(target, objective string, task TaskSpec, subtask SubtaskSpec) string {
	if isWebTarget(target) {
		if shouldUseSQLValidation(target, objective) {
			return buildSQLMapCommand(target)
		}
		if shouldUseContentDiscovery(objective) {
			return buildFFUFCommand(target)
		}
		if shouldUseTemplateScan(objective) {
			return buildNucleiCommand(target)
		}
		return buildBaselineCommand(target)
	}
	host := targetHost(target)
	if host == "" {
		host = strings.TrimSpace(target)
	}
	if host == "" {
		return ""
	}
	return fmt.Sprintf("printf '%%s\\n' %s | dnsx -silent -resp-only", shellQuote(host))
}

func composeResearchQuery(target, objective string) string {
	return strings.TrimSpace(strings.Join([]string{target, objective, "technology stack authentication admin exposure research"}, " "))
}

func composeCodeQuery(target, objective string) string {
	return strings.TrimSpace(strings.Join([]string{target, objective, "api docs source repository code reference manual"}, " "))
}

func composeExploitQuery(target, objective string) string {
	return strings.TrimSpace(strings.Join([]string{target, objective, "cve exploit poc advisory vulnerability"}, " "))
}

func buildBaselineCommand(target string) string {
	if isWebTarget(target) {
		return fmt.Sprintf("httpx -u %s -title -tech-detect -status-code -follow-redirects -silent", shellQuote(target))
	}
	host := targetHost(target)
	if host == "" {
		host = strings.TrimSpace(target)
	}
	return fmt.Sprintf("printf '%%s\\n' %s | dnsx -silent -resp-only", shellQuote(host))
}

func buildFFUFCommand(target string) string {
	return fmt.Sprintf("printf 'admin\\nlogin\\napi\\nbackup\\nrobots.txt\\n' | ffuf -u %s -w - -mc all -fc 404", shellQuote(strings.TrimRight(target, "/")+"/FUZZ"))
}

func buildNucleiCommand(target string) string {
	return fmt.Sprintf("printf '%%s\\n' %s | nuclei -silent -nc -rl 2", shellQuote(target))
}

func buildSQLMapCommand(target string) string {
	return fmt.Sprintf("sqlmap -u %s --batch --random-agent --level 1 --risk 1 --threads 1 --smart", shellQuote(target))
}

func shouldUseContentDiscovery(objective string) bool {
	return containsAny(objective, "admin", "login", "panel", "directory", "content", "path", "route", "hidden")
}

func shouldUseTemplateScan(objective string) bool {
	return containsAny(objective, "vulnerability", "vulnerabilities", "cve", "exposure", "misconfig", "template", "nuclei")
}

func shouldUseSQLValidation(target, objective string) bool {
	return strings.Contains(target, "?") || containsAny(objective, "sql", "sqli", "sql injection", "parameter", "query", "id=")
}

func containsAny(value string, candidates ...string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	for _, candidate := range candidates {
		if strings.Contains(lower, candidate) {
			return true
		}
	}
	return false
}

func isWebTarget(target string) bool {
	lower := strings.ToLower(strings.TrimSpace(target))
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

func targetHost(target string) string {
	parsed, err := url.Parse(strings.TrimSpace(target))
	if err == nil && parsed.Hostname() != "" {
		return parsed.Hostname()
	}
	return strings.TrimSpace(target)
}

func shellQuote(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(trimmed, "'", `'"'"'`) + "'"
}

func renderPlan(plan FlowPlan) string {
	var b strings.Builder
	for _, task := range plan.Tasks {
		fmt.Fprintf(&b, "- task: %s [%s]\n", task.Name, task.Role)
		if task.Description != "" {
			fmt.Fprintf(&b, "  description: %s\n", task.Description)
		}
		for _, subtask := range task.Subtasks {
			fmt.Fprintf(&b, "  - subtask: %s [%s]\n", subtask.Name, subtask.Role)
			if subtask.Description != "" {
				fmt.Fprintf(&b, "    description: %s\n", subtask.Description)
			}
			fmt.Fprintf(&b, "    attempts: %d | escalate_to: %s\n", subtask.MaxAttempts, subtask.EscalateTo)
			for _, step := range subtask.Steps {
				fmt.Fprintf(&b, "    - step: %s\n", step.FunctionName)
				for key, value := range step.Input {
					fmt.Fprintf(&b, "      %s: %s\n", key, value)
				}
			}
		}
	}
	return strings.TrimSpace(b.String())
}
