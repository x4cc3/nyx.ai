package httpapi

import (
	"html/template"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"nyx/internal/domain"
)

var workspaceIndexTemplate = template.Must(template.New("workspace-index").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>NYX Workspace</title>
  <style>
    :root { color-scheme: light; --bg:#f5efe5; --ink:#1f1b18; --panel:#fffaf3; --line:#dbcdb8; --accent:#8f3d2e; --muted:#6d5c4e; }
    body { margin:0; font-family: Georgia, "Iowan Old Style", serif; background:linear-gradient(180deg,#f7f1e8,#efe4d2); color:var(--ink); }
    main { max-width: 980px; margin: 0 auto; padding: 40px 20px 80px; }
    h1,h2 { margin:0 0 12px; }
    .hero { margin-bottom: 24px; }
    .grid { display:grid; gap:16px; }
    .card { background:var(--panel); border:1px solid var(--line); border-radius:18px; padding:18px; box-shadow:0 12px 28px rgba(31,27,24,.06); }
    .meta { color:var(--muted); font-size:14px; }
    a { color:var(--accent); text-decoration:none; }
    .pill { display:inline-block; padding:4px 10px; border-radius:999px; background:#efe2d1; color:var(--accent); font-size:12px; }
  </style>
</head>
<body>
<main>
  <section class="hero">
    <h1>NYX Workspace</h1>
    <p class="meta">Tenant: {{ .TenantID }} | Queue: {{ .QueueMode }} | Approvals pending: {{ .PendingApprovals }}</p>
  </section>
  <section class="grid">
    {{ range .Flows }}
    <article class="card">
      <div class="pill">{{ .Status }}</div>
      <h2><a href="/workspace/flows/{{ .ID }}">{{ .Name }}</a></h2>
      <p>{{ .Objective }}</p>
      <p class="meta">{{ .Target }} | {{ .CreatedAt.Format "2006-01-02 15:04:05 MST" }}</p>
    </article>
    {{ else }}
    <article class="card">
      <h2>No Flows Yet</h2>
      <p class="meta">Create a flow through the API to populate the operator workspace.</p>
    </article>
    {{ end }}
  </section>
</main>
</body>
</html>`))

var workspaceFlowTemplate = template.Must(template.New("workspace-flow").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{ .Flow.Name }} | NYX Workspace</title>
  <style>
    :root { color-scheme: light; --bg:#f0e7da; --panel:#fff9f0; --line:#d6c5af; --ink:#211b16; --muted:#6c5a4a; --accent:#9d412f; }
    body { margin:0; font-family: Georgia, "Iowan Old Style", serif; background:radial-gradient(circle at top left,#fbf4ea,#eee1cf 58%); color:var(--ink); }
    main { max-width: 1180px; margin:0 auto; padding: 32px 20px 80px; }
    header { margin-bottom: 24px; }
    h1,h2 { margin:0 0 10px; }
    .meta { color:var(--muted); font-size:14px; }
    .grid { display:grid; gap:16px; grid-template-columns: repeat(auto-fit,minmax(260px,1fr)); }
    .panel { background:var(--panel); border:1px solid var(--line); border-radius:18px; padding:18px; box-shadow:0 10px 24px rgba(33,27,22,.05); }
    ul { margin:0; padding-left:18px; }
    li { margin:0 0 8px; }
    code { font-family: "IBM Plex Mono", monospace; font-size: 12px; }
    .pill { display:inline-block; padding:4px 10px; border-radius:999px; background:#eddcc8; color:var(--accent); font-size:12px; }
    a { color:var(--accent); text-decoration:none; }
  </style>
</head>
<body>
<main>
  <header>
    <p><a href="/workspace">Back to workspace</a></p>
    <div class="pill">{{ .Flow.Status }}</div>
    <h1>{{ .Flow.Name }}</h1>
    <p>{{ .Flow.Objective }}</p>
    <p class="meta">Tenant: {{ .TenantID }} | Queue: {{ .QueueMode }} | Target: {{ .Flow.Target }}</p>
  </header>
  <section class="grid">
    <article class="panel">
      <h2>Approvals</h2>
      <ul>{{ range .Approvals }}<li><strong>{{ .Status }}</strong> {{ .Kind }} by {{ .RequestedBy }}</li>{{ else }}<li>No approvals recorded.</li>{{ end }}</ul>
    </article>
    <article class="panel">
      <h2>Tasks</h2>
      <ul>{{ range .Tasks }}<li><strong>{{ .Name }}</strong> <span class="meta">{{ .AgentRole }} / {{ .Status }}</span></li>{{ else }}<li>No tasks yet.</li>{{ end }}</ul>
    </article>
    <article class="panel">
      <h2>Subtasks</h2>
      <ul>{{ range .Subtasks }}<li><strong>{{ .Name }}</strong> <span class="meta">{{ .AgentRole }} / {{ .Status }}</span></li>{{ else }}<li>No subtasks yet.</li>{{ end }}</ul>
    </article>
    <article class="panel">
      <h2>Actions</h2>
      <ul>{{ range .Actions }}<li><strong>{{ .FunctionName }}</strong> <span class="meta">{{ .AgentRole }} / {{ .Status }}</span></li>{{ else }}<li>No actions yet.</li>{{ end }}</ul>
    </article>
    <article class="panel">
      <h2>Findings</h2>
      <ul>{{ range .Findings }}<li><strong>{{ .Title }}</strong> <span class="meta">{{ .Severity }}</span></li>{{ else }}<li>No findings yet.</li>{{ end }}</ul>
    </article>
    <article class="panel">
      <h2>Executions</h2>
      <ul>
        {{ range .Executions }}
        <li>
          <strong>{{ .Profile }}</strong> <span class="meta">{{ .Runtime }} / {{ .Status }}</span>
          {{ with index .Metadata "image" }}<br><span class="meta">image:</span> <code>{{ . }}</code>{{ end }}
          {{ with index .Metadata "network_mode" }}<br><span class="meta">network:</span> <code>{{ . }}</code>{{ end }}
          {{ with index .Metadata "command" }}<br><span class="meta">command:</span> <code>{{ . }}</code>{{ end }}
          {{ with index .Metadata "evidence_paths" }}<br><span class="meta">evidence:</span> <code>{{ . }}</code>{{ end }}
        </li>
        {{ else }}
        <li>No executions recorded.</li>
        {{ end }}
      </ul>
    </article>
    <article class="panel">
      <h2>Artifacts</h2>
      <ul>
        {{ range .Artifacts }}
        <li>
          <strong>{{ .Name }}</strong> <span class="meta">{{ .Kind }}</span>
          <br><code>{{ .Content }}</code>
          {{ with index .Metadata "image" }}<br><span class="meta">image:</span> <code>{{ . }}</code>{{ end }}
          {{ with index .Metadata "network_mode" }}<br><span class="meta">network:</span> <code>{{ . }}</code>{{ end }}
          {{ with index .Metadata "command" }}<br><span class="meta">command:</span> <code>{{ . }}</code>{{ end }}
          {{ with index .Metadata "evidence_paths" }}<br><span class="meta">evidence:</span> <code>{{ . }}</code>{{ end }}
        </li>
        {{ else }}
        <li>No artifacts yet.</li>
        {{ end }}
      </ul>
    </article>
  </section>
</main>
</body>
</html>`))

var workspaceBuildRequiredTemplate = template.Must(template.New("workspace-build-required").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>NYX Workspace Unavailable</title>
  <style>
    body { margin:0; min-height:100vh; display:grid; place-items:center; background:#171210; color:#f4ebdf; font-family:"Iowan Old Style", Georgia, serif; }
    main { max-width:700px; padding:32px; border:1px solid rgba(244,106,65,.22); border-radius:24px; background:rgba(35,25,22,.94); box-shadow:0 20px 50px rgba(0,0,0,.35); }
    h1 { margin:0 0 12px; font-family:"Avenir Next Condensed","Franklin Gothic Medium",sans-serif; }
    p { line-height:1.6; color:#ceb9a3; }
    code { color:#ffb281; }
  </style>
</head>
<body>
<main>
  <h1>Workspace Frontend Required</h1>
  <p>This deployment has API-key protection enabled, so the legacy server-rendered workspace is disabled.</p>
  <p>The primary operator UI now lives in the standalone Next app under <code>web/</code>. This <code>/workspace</code> route is a legacy fallback that only works when the older workspace bundle has been built.</p>
</main>
</body>
</html>`))

// workspaceDistDir is a legacy compatibility artifact for the older Go-served
// /workspace UI. The primary operator UI is the standalone Next app in web/.
const workspaceDistDir = "web/dist"

func (s *Server) handleWorkspaceRoutes(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/workspace/assets/") {
		if s.serveWorkspaceAsset(w, r) {
			return
		}
		http.NotFound(w, r)
		return
	}

	switch {
	case r.URL.Path == "/workspace", r.URL.Path == "/workspace/":
		s.handleWorkspaceIndex(w, r)
	case strings.HasPrefix(r.URL.Path, "/workspace/flows/"):
		s.handleWorkspaceFlow(w, r)
	default:
		if s.serveWorkspaceApp(w, r) {
			return
		}
		http.NotFound(w, r)
	}
}

func (s *Server) handleWorkspaceIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if s.serveWorkspaceApp(w, r) {
		return
	}
	if s.cfg.APIKey != "" {
		s.renderWorkspaceBuildRequired(w)
		return
	}
	tenantID := currentTenant(r, s.cfg.DefaultTenant)
	ctx := r.Context()
	flows, err := s.repo.ListFlowsByTenant(ctx, tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list_flows_failed", err.Error())
		return
	}
	approvals, _ := s.repo.ListApprovalsByTenant(ctx, tenantID)
	pending := 0
	for _, approval := range approvals {
		if approval.Status == domain.ApprovalStatusPending {
			pending++
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = workspaceIndexTemplate.Execute(w, map[string]any{
		"TenantID":         tenantID,
		"QueueMode":        s.queue.Mode(),
		"Flows":            flows,
		"PendingApprovals": pending,
	})
}

func (s *Server) handleWorkspaceFlow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if s.serveWorkspaceApp(w, r) {
		return
	}
	if s.cfg.APIKey != "" {
		s.renderWorkspaceBuildRequired(w)
		return
	}
	flowID := strings.Trim(strings.TrimPrefix(r.URL.Path, "/workspace/flows/"), "/")
	if flowID == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	workspace, err := s.workspacePayload(r.Context(), currentTenant(r, s.cfg.DefaultTenant), flowID)
	if err != nil {
		writeError(w, http.StatusNotFound, "flow_not_found", "Flow not found")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = workspaceFlowTemplate.Execute(w, workspace)
}

func (s *Server) serveWorkspaceApp(w http.ResponseWriter, r *http.Request) bool {
	indexPath := filepath.Join(workspaceDistDir, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		return false
	}
	http.ServeFile(w, r, indexPath)
	return true
}

func (s *Server) serveWorkspaceAsset(w http.ResponseWriter, r *http.Request) bool {
	root, err := filepath.Abs(workspaceDistDir)
	if err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(root, "index.html")); err != nil {
		return false
	}

	cleaned := path.Clean(strings.TrimPrefix(r.URL.Path, "/workspace/"))
	if cleaned == "." || cleaned == "" || strings.HasPrefix(cleaned, "..") {
		return false
	}
	assetPath := filepath.Join(root, filepath.FromSlash(cleaned))
	if assetPath != root && !strings.HasPrefix(assetPath, root+string(os.PathSeparator)) {
		return false
	}
	if _, err := os.Stat(assetPath); err != nil {
		return false
	}
	http.ServeFile(w, r, assetPath)
	return true
}

func (s *Server) renderWorkspaceBuildRequired(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusServiceUnavailable)
	_ = workspaceBuildRequiredTemplate.Execute(w, nil)
}
