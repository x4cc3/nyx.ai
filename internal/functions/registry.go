package functions

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"nyx/internal/domain"
	"nyx/internal/executor"
	"nyx/internal/services/browser"
	"nyx/internal/services/search"
)

var hrefPattern = regexp.MustCompile(`(?is)href=["']([^"'#]+)["']`)

func (g *Gateway) buildRegistry() []ToolSpec {
	return []ToolSpec{
		{
			Definition: domain.FunctionDef{
				Name:          "done",
				Description:   "Mark a specialist subtask complete.",
				Profile:       "control",
				Category:      "barrier",
				SafetyProfile: "control",
			},
			Handler: func(_ context.Context, _, _ string, _ map[string]string) CallResult {
				return CallResult{
					Profile: "control",
					Runtime: "go-orchestrator",
					Output:  map[string]string{"summary": "Subtask marked complete."},
				}
			},
		},
		{
			Definition: domain.FunctionDef{
				Name:          "ask",
				Description:   "Request operator input when execution is blocked.",
				Profile:       "control",
				Category:      "barrier",
				SafetyProfile: "control",
				InputSchema: []domain.FunctionInputField{
					{Name: "message", Type: "string", Description: "Operator-facing question or escalation note.", Required: false},
					{Name: "reason", Type: "string", Description: "Structured reason for the escalation.", Required: false},
				},
			},
			Handler: func(_ context.Context, _, _ string, _ map[string]string) CallResult {
				return CallResult{
					Profile: "control",
					Runtime: "go-orchestrator",
					Output:  map[string]string{"summary": "Operator input requested."},
				}
			},
		},
		{
			Definition: domain.FunctionDef{
				Name:          "terminal",
				Description:   "Execute terminal work inside an isolated execution profile.",
				Profile:       "terminal",
				Category:      "environment",
				SafetyProfile: "scoped",
				InputSchema: []domain.FunctionInputField{
					{Name: "target", Type: "string", Description: "Authorized target URL or hostname for scope validation.", Required: false},
					{Name: "command", Type: "string", Description: "Shell command to execute.", Required: false},
					{Name: "goal", Type: "string", Description: "High-level execution goal when no explicit command is given.", Required: false},
					{Name: "toolset", Type: "enum(general|pentest)", Description: "Requested worker toolset.", Required: false},
				},
			},
			Handler: g.handleTerminal,
		},
		{
			Definition: domain.FunctionDef{
				Name:                 "terminal_exec",
				Description:          "Execute a first-class terminal command with pentest-capable worker selection.",
				Profile:              "terminal",
				Category:             "environment",
				SafetyProfile:        "scoped",
				RequiresPentestImage: true,
				InputSchema: []domain.FunctionInputField{
					{Name: "target", Type: "string", Description: "Authorized target URL or hostname for scope validation.", Required: true},
					{Name: "command", Type: "string", Description: "Shell command to execute.", Required: true},
					{Name: "goal", Type: "string", Description: "High-level intent for operator-visible metadata.", Required: false},
				},
			},
			Handler: func(ctx context.Context, flowID, actionID string, input map[string]string) CallResult {
				next := cloneInput(input)
				if strings.TrimSpace(next["toolset"]) == "" {
					next["toolset"] = "pentest"
				}
				return g.handleTerminal(ctx, flowID, actionID, next)
			},
		},
		{
			Definition: domain.FunctionDef{
				Name:          "file",
				Description:   "Read or write artifacts inside an isolated execution profile.",
				Profile:       "file",
				Category:      "environment",
				SafetyProfile: "workspace",
				InputSchema: []domain.FunctionInputField{
					{Name: "operation", Type: "enum(write|read|list)", Description: "File action to perform inside the workspace.", Required: false},
					{Name: "path", Type: "string", Description: "Workspace-relative path.", Required: false},
					{Name: "content", Type: "string", Description: "Content to write for write operations.", Required: false},
				},
			},
			Handler: g.handleFile,
		},
		{
			Definition: domain.FunctionDef{
				Name:          "file_read",
				Description:   "Read a workspace-relative file through the isolated file profile.",
				Profile:       "file",
				Category:      "environment",
				SafetyProfile: "workspace",
				InputSchema: []domain.FunctionInputField{
					{Name: "path", Type: "string", Description: "Workspace-relative path to read.", Required: true},
				},
			},
			Handler: func(ctx context.Context, flowID, actionID string, input map[string]string) CallResult {
				next := cloneInput(input)
				next["operation"] = "read"
				return g.handleFile(ctx, flowID, actionID, next)
			},
		},
		{
			Definition: domain.FunctionDef{
				Name:          "file_write",
				Description:   "Write content to a workspace-relative file through the isolated file profile.",
				Profile:       "file",
				Category:      "environment",
				SafetyProfile: "workspace",
				InputSchema: []domain.FunctionInputField{
					{Name: "path", Type: "string", Description: "Workspace-relative path to write.", Required: true},
					{Name: "content", Type: "string", Description: "Content to write to the file.", Required: true},
				},
			},
			Handler: func(ctx context.Context, flowID, actionID string, input map[string]string) CallResult {
				next := cloneInput(input)
				next["operation"] = "write"
				return g.handleFile(ctx, flowID, actionID, next)
			},
		},
		{
			Definition: domain.FunctionDef{
				Name:            "browser",
				Description:     "Navigate and inspect target pages with the Go browser service.",
				Profile:         "browser",
				Category:        "search_network",
				SafetyProfile:   "scoped",
				RequiresNetwork: true,
				InputSchema: []domain.FunctionInputField{
					{Name: "target", Type: "string", Description: "Authorized target URL or hostname for scope validation.", Required: true},
					{Name: "url", Type: "string", Description: "Page URL to visit.", Required: true},
					{Name: "wait_selector", Type: "string", Description: "Optional selector to wait for before capturing content.", Required: false},
				},
			},
			Handler: g.handleBrowser,
		},
		{
			Definition: domain.FunctionDef{
				Name:            "browser_html",
				Description:     "Capture rendered HTML and snapshot metadata from an in-scope target page.",
				Profile:         "browser",
				Category:        "search_network",
				SafetyProfile:   "scoped",
				RequiresNetwork: true,
				InputSchema: []domain.FunctionInputField{
					{Name: "target", Type: "string", Description: "Authorized target URL or hostname for scope validation.", Required: true},
					{Name: "url", Type: "string", Description: "Page URL to visit.", Required: true},
				},
			},
			Handler: func(ctx context.Context, flowID, actionID string, input map[string]string) CallResult {
				return g.handleBrowserVariant(ctx, flowID, actionID, input, "html")
			},
		},
		{
			Definition: domain.FunctionDef{
				Name:            "browser_markdown",
				Description:     "Capture normalized markdown-like page content from an in-scope target page.",
				Profile:         "browser",
				Category:        "search_network",
				SafetyProfile:   "scoped",
				RequiresNetwork: true,
				InputSchema: []domain.FunctionInputField{
					{Name: "target", Type: "string", Description: "Authorized target URL or hostname for scope validation.", Required: true},
					{Name: "url", Type: "string", Description: "Page URL to visit.", Required: true},
				},
			},
			Handler: func(ctx context.Context, flowID, actionID string, input map[string]string) CallResult {
				return g.handleBrowserVariant(ctx, flowID, actionID, input, "markdown")
			},
		},
		{
			Definition: domain.FunctionDef{
				Name:            "browser_links",
				Description:     "Extract and normalize visible links from an in-scope target page.",
				Profile:         "browser",
				Category:        "search_network",
				SafetyProfile:   "scoped",
				RequiresNetwork: true,
				InputSchema: []domain.FunctionInputField{
					{Name: "target", Type: "string", Description: "Authorized target URL or hostname for scope validation.", Required: true},
					{Name: "url", Type: "string", Description: "Page URL to visit.", Required: true},
				},
			},
			Handler: func(ctx context.Context, flowID, actionID string, input map[string]string) CallResult {
				return g.handleBrowserVariant(ctx, flowID, actionID, input, "links")
			},
		},
		{
			Definition: domain.FunctionDef{
				Name:            "browser_screenshot",
				Description:     "Capture a rendered screenshot from an in-scope target page without saving full HTML content.",
				Profile:         "browser",
				Category:        "search_network",
				SafetyProfile:   "scoped",
				RequiresNetwork: true,
				InputSchema: []domain.FunctionInputField{
					{Name: "target", Type: "string", Description: "Authorized target URL or hostname for scope validation.", Required: true},
					{Name: "url", Type: "string", Description: "Page URL to visit.", Required: true},
					{Name: "wait_selector", Type: "string", Description: "Optional selector to wait for before capturing the screenshot.", Required: false},
				},
			},
			Handler: func(ctx context.Context, flowID, actionID string, input map[string]string) CallResult {
				return g.handleBrowserVariant(ctx, flowID, actionID, input, "screenshot")
			},
		},
		{
			Definition: domain.FunctionDef{
				Name:            "search_web",
				Description:     "Gather external target context and recon hints.",
				Profile:         "search",
				Category:        "search_network",
				SafetyProfile:   "recon",
				RequiresNetwork: true,
				InputSchema: []domain.FunctionInputField{
					{Name: "target", Type: "string", Description: "Authorized target or hostname to research.", Required: false},
					{Name: "objective", Type: "string", Description: "Current mission objective.", Required: false},
					{Name: "query", Type: "string", Description: "Explicit search query override.", Required: false},
				},
			},
			Handler: g.handleSearchWeb,
		},
		{
			Definition: domain.FunctionDef{
				Name:            "search_deep",
				Description:     "Run a deeper public research query over an authorized target.",
				Profile:         "search",
				Category:        "search_network",
				SafetyProfile:   "recon",
				RequiresNetwork: true,
				InputSchema: []domain.FunctionInputField{
					{Name: "target", Type: "string", Description: "Authorized target or hostname to research.", Required: false},
					{Name: "objective", Type: "string", Description: "Current mission objective.", Required: false},
					{Name: "query", Type: "string", Description: "Deep-research query to execute.", Required: false},
				},
			},
			Handler: func(ctx context.Context, flowID, actionID string, input map[string]string) CallResult {
				next := cloneInput(input)
				next["query"] = composeSearchVariantQuery(input, "deep")
				return g.handleSearch(ctx, next, search.KindDeep)
			},
		},
		{
			Definition: domain.FunctionDef{
				Name:            "search_exploits",
				Description:     "Search public exploit and PoC references tied to the current target or finding.",
				Profile:         "search",
				Category:        "search_network",
				SafetyProfile:   "recon",
				RequiresNetwork: true,
				InputSchema: []domain.FunctionInputField{
					{Name: "target", Type: "string", Description: "Authorized target or hostname to research.", Required: false},
					{Name: "objective", Type: "string", Description: "Current mission objective or suspected issue.", Required: false},
					{Name: "query", Type: "string", Description: "Exploit-oriented query to execute.", Required: false},
				},
			},
			Handler: func(ctx context.Context, flowID, actionID string, input map[string]string) CallResult {
				next := cloneInput(input)
				next["query"] = composeSearchVariantQuery(input, "exploit")
				return g.handleSearch(ctx, next, search.KindExploit)
			},
		},
		{
			Definition: domain.FunctionDef{
				Name:            "search_code",
				Description:     "Search public code snippets, manuals, and operator-facing references tied to the current target.",
				Profile:         "search",
				Category:        "search_network",
				SafetyProfile:   "recon",
				RequiresNetwork: true,
				InputSchema: []domain.FunctionInputField{
					{Name: "target", Type: "string", Description: "Authorized target or hostname to research.", Required: false},
					{Name: "objective", Type: "string", Description: "Current mission objective or implementation clue.", Required: false},
					{Name: "query", Type: "string", Description: "Code or manual query to execute.", Required: false},
				},
			},
			Handler: func(ctx context.Context, flowID, actionID string, input map[string]string) CallResult {
				next := cloneInput(input)
				next["query"] = composeSearchVariantQuery(input, "code")
				return g.handleSearch(ctx, next, search.KindCode)
			},
		},
		{
			Definition: domain.FunctionDef{
				Name:          "search_memory",
				Description:   "Search the NYX memory layer for target observations, exploit references, manuals, or operator notes.",
				Profile:       "memory",
				Category:      "search_vector_db",
				SafetyProfile: "memory",
				InputSchema: []domain.FunctionInputField{
					{Name: "query", Type: "string", Description: "Search phrase for prior NYX memory entries.", Required: true},
					{Name: "namespace", Type: "enum(target_observations|exploit_references|reference_materials|operator_notes|all)", Description: "Optional memory namespace filter.", Required: false},
				},
			},
			Handler: g.handleSearchMemory,
		},
	}
}

func (g *Gateway) handleSearchWeb(ctx context.Context, _ string, _ string, input map[string]string) CallResult {
	return g.handleSearch(ctx, input, search.KindWeb)
}

func (g *Gateway) handleSearch(ctx context.Context, input map[string]string, kind string) CallResult {
	result, err := g.search.Search(ctx, search.Request{
		Target:    input["target"],
		Objective: input["objective"],
		Query:     input["query"],
		Kind:      kind,
	})
	output := map[string]string{
		"query":   result.Query,
		"source":  result.Source,
		"summary": result.Summary,
	}
	for idx, item := range result.Results {
		key := strconv.Itoa(idx + 1)
		output["result_"+key+"_title"] = item.Title
		output["result_"+key+"_url"] = item.URL
		if item.Snippet != "" {
			output["result_"+key+"_snippet"] = item.Snippet
		}
	}
	callResult := CallResult{
		Profile: "search",
		Runtime: "go-search-service",
		Output:  output,
		Err:     err,
	}
	if callResult.Output["summary"] == "" {
		callResult.Output["summary"] = fmt.Sprintf("Search failed for %s.", input["target"])
	}
	if err == nil && callResult.Output["summary"] == "" {
		callResult.Output["summary"] = fmt.Sprintf("Collected web context for %s.", input["target"])
	}
	return callResult
}

func (g *Gateway) handleSearchMemory(ctx context.Context, flowID, _ string, input map[string]string) CallResult {
	namespace := strings.TrimSpace(input["namespace"])
	memories := g.memory.SearchNamespace(ctx, flowID, input["query"], namespace)
	parts := make([]string, 0, len(memories))
	for _, item := range memories {
		content := item.Content
		if ns := domain.MemoryNamespace(item.Kind, item.Metadata); ns != "" {
			content = "[" + ns + "] " + content
		}
		parts = append(parts, content)
	}
	return CallResult{
		Profile: "memory",
		Runtime: "go-memory-service",
		Output: map[string]string{
			"matches":   fmt.Sprintf("%d", len(memories)),
			"namespace": blankNamespace(namespace),
			"summary":   strings.Join(parts, " | "),
		},
	}
}

func blankNamespace(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return domain.MemoryNamespaceAll
	}
	return value
}

func (g *Gateway) handleBrowser(ctx context.Context, _ string, _ string, input map[string]string) CallResult {
	if err := validateBrowserScope(input["target"], input["url"]); err != nil {
		return CallResult{
			Profile: "browser",
			Runtime: "go-browser-service",
			Output:  map[string]string{"summary": err.Error()},
			Err:     err,
		}
	}
	result := g.browser.Navigate(ctx, browser.Request{
		URL:                input["url"],
		WaitSelector:       input["wait_selector"],
		AuthHeader:         input["auth_header"],
		CookiesJSON:        input["cookies_json"],
		LocalStorageJSON:   input["local_storage_json"],
		SessionStorageJSON: input["session_storage_json"],
		CaptureMode:        input["capture_mode"],
	})
	return CallResult{
		Profile: "browser",
		Runtime: "go-browser-service",
		Output: map[string]string{
			"title":           result.Title,
			"summary":         result.Summary,
			"final_url":       result.FinalURL,
			"text":            result.Text,
			"html":            result.HTML,
			"mode":            result.Mode,
			"auth_state":      result.AuthState,
			"screenshot_path": result.ScreenshotPath,
			"snapshot_path":   result.SnapshotPath,
			"status_code":     strconv.Itoa(result.StatusCode),
		},
	}
}

func (g *Gateway) handleBrowserVariant(ctx context.Context, flowID, actionID string, input map[string]string, variant string) CallResult {
	next := cloneInput(input)
	if variant == "screenshot" {
		next["capture_mode"] = "screenshot"
	}
	base := g.handleBrowser(ctx, flowID, actionID, next)
	if base.Err != nil {
		return base
	}

	switch variant {
	case "html":
		base.Output["summary"] = summarizeVariant(base.Output["final_url"], "Captured rendered HTML for operator review.")
	case "markdown":
		base.Output["markdown"] = toMarkdown(base.Output["title"], base.Output["text"], base.Output["final_url"])
		base.Output["summary"] = summarizeVariant(base.Output["final_url"], "Captured normalized markdown content.")
	case "links":
		links := extractLinks(base.Output["html"], base.Output["final_url"])
		base.Output["link_count"] = strconv.Itoa(len(links))
		for idx, link := range links {
			base.Output[fmt.Sprintf("link_%d_url", idx+1)] = link
		}
		base.Output["summary"] = summarizeVariant(base.Output["final_url"], fmt.Sprintf("Extracted %d links from the rendered page.", len(links)))
	case "screenshot":
		delete(base.Output, "html")
		delete(base.Output, "text")
		delete(base.Output, "snapshot_path")
		base.Output["summary"] = summarizeVariant(base.Output["final_url"], "Captured a rendered screenshot for operator review.")
	}
	return base
}

func (g *Gateway) handleTerminal(ctx context.Context, flowID, actionID string, input map[string]string) CallResult {
	if err := validateCommandScope(input["target"], input["command"]); err != nil {
		return CallResult{
			Profile: "terminal",
			Runtime: g.exec.Mode(),
			Output:  map[string]string{"summary": err.Error()},
			Err:     err,
		}
	}
	result, err := g.exec.Execute(ctx, executor.Request{
		FlowID:       flowID,
		ActionID:     actionID,
		Profile:      "terminal",
		FunctionName: "terminal",
		Input:        input,
	})
	return CallResult{
		Profile: "terminal",
		Runtime: result.Runtime,
		Output:  executorOutput(result),
		Err:     err,
	}
}

func (g *Gateway) handleFile(ctx context.Context, flowID, actionID string, input map[string]string) CallResult {
	result, err := g.exec.Execute(ctx, executor.Request{
		FlowID:       flowID,
		ActionID:     actionID,
		Profile:      "file",
		FunctionName: "file",
		Input:        input,
	})
	return CallResult{
		Profile: "file",
		Runtime: result.Runtime,
		Output:  executorOutput(result),
		Err:     err,
	}
}

func composeSearchVariantQuery(input map[string]string, variant string) string {
	base := strings.TrimSpace(input["query"])
	if base == "" {
		base = strings.TrimSpace(strings.Join([]string{input["target"], input["objective"]}, " "))
	}
	switch variant {
	case "exploit":
		return strings.TrimSpace(base + " exploit PoC CVE advisory")
	case "deep":
		return strings.TrimSpace(base + " architecture exposure dependencies authentication research")
	case "code":
		return strings.TrimSpace(base + " source code repository docs manual api reference")
	default:
		return base
	}
}

func toMarkdown(title, text, finalURL string) string {
	var b strings.Builder
	if strings.TrimSpace(title) != "" {
		b.WriteString("# ")
		b.WriteString(strings.TrimSpace(title))
		b.WriteString("\n\n")
	}
	if strings.TrimSpace(finalURL) != "" {
		b.WriteString("- URL: ")
		b.WriteString(strings.TrimSpace(finalURL))
		b.WriteString("\n\n")
	}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		b.WriteString("- ")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return strings.TrimSpace(b.String())
}

func extractLinks(htmlBody, baseURL string) []string {
	if strings.TrimSpace(htmlBody) == "" {
		return nil
	}
	base, _ := url.Parse(strings.TrimSpace(baseURL))
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, match := range hrefPattern.FindAllStringSubmatch(htmlBody, -1) {
		if len(match) < 2 {
			continue
		}
		raw := strings.TrimSpace(match[1])
		if raw == "" {
			continue
		}
		parsed, err := url.Parse(raw)
		if err != nil {
			continue
		}
		if base != nil {
			parsed = base.ResolveReference(parsed)
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			continue
		}
		normalized := parsed.String()
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	sort.Strings(out)
	return out
}

func summarizeVariant(finalURL, summary string) string {
	if strings.TrimSpace(finalURL) == "" {
		return summary
	}
	return strings.TrimSpace(summary + " " + finalURL)
}

func cloneInput(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
