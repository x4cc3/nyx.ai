import type {
  Approval,
  ArchitectureResponse,
  Flow,
  FlowEvent,
  LiveChatResponse,
  ReportSummaryResponse,
  ScanAssistResponse,
  ScanFinding,
  ScanReport,
  ScanStatusData,
  ScanStep,
  ScanSummary,
  ToolStreamEvent,
  Workspace,
  AgentGraph,
} from "./types";

const API_BASE = "/api";

export class ApiError extends Error {
  status: number;
  code?: string;
  fieldErrors?: Record<string, string> | null;

  constructor(
    message: string,
    status: number,
    code?: string,
    fieldErrors?: Record<string, string> | null,
  ) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.fieldErrors = fieldErrors ?? null;
  }
}

type ApiErrorPayload = {
  error?: {
    code?: string;
    message?: string;
    field_errors?: Record<string, string> | null;
  };
};

async function apiFetch<T>(path: string, options?: RequestInit): Promise<T> {
  const defaultHeaders: Record<string, string> = {};
  if (options?.body && typeof options.body === "string") {
    defaultHeaders["Content-Type"] = "application/json";
  }

  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 30000);
  try {
    const response = await fetch(`${API_BASE}${path}`, {
      ...options,
      signal: controller.signal,
      headers: {
        ...defaultHeaders,
        ...(options?.headers || {}),
      },
      cache: "no-store",
    });

    if (!response.ok) {
      let payload: ApiErrorPayload | null = null;
      try {
        payload = (await response.json()) as ApiErrorPayload;
      } catch {
        payload = null;
      }

      throw new ApiError(
        payload?.error?.message || `API error ${response.status}`,
        response.status,
        payload?.error?.code,
        payload?.error?.field_errors ?? null,
      );
    }

    return (await response.json()) as T;
  } finally {
    clearTimeout(timeout);
  }
}

export async function getArchitecture() {
  return apiFetch<ArchitectureResponse>("/v1/architecture");
}

export async function listFlows() {
  return apiFetch<{ flows: Flow[] }>("/v1/flows");
}

export async function listApprovals() {
  return apiFetch<{ approvals: Approval[] }>("/v1/approvals");
}

export async function createFlow(payload: {
  name: string;
  target: string;
  objective: string;
}) {
  return apiFetch<Flow>("/v1/flows", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function startFlow(flowId: string) {
  return apiFetch<{
    status: string;
    flow_id: string;
    queue_mode?: string;
    approval_id?: string;
  }>(`/v1/flows/${flowId}/start`, {
    method: "POST",
  });
}

export async function getFlow(flowId: string) {
  return apiFetch<Flow>(`/v1/flows/${flowId}`);
}

export async function getWorkspace(flowId: string) {
  return apiFetch<Workspace>(`/v1/flows/${flowId}/workspace`);
}

async function listWorkspaces(flowIds: string[]) {
  if (flowIds.length === 0) {
    return { workspaces: [] as Workspace[], errors: {} as Record<string, string> };
  }
  const params = new URLSearchParams({
    flow_ids: flowIds.join(","),
  });
  return apiFetch<{ workspaces: Workspace[]; errors?: Record<string, string> }>(
    `/v1/workspaces?${params.toString()}`,
  );
}

function summarizeFindings(findings: Workspace["findings"]) {
  return findings.reduce(
    (acc, finding) => {
      const key = String(finding.severity || "info").toLowerCase();
      if (
        key === "critical" ||
        key === "high" ||
        key === "medium" ||
        key === "low" ||
        key === "info"
      ) {
        acc[key] += 1;
      }
      return acc;
    },
    {
      critical: 0,
      high: 0,
      medium: 0,
      low: 0,
      info: 0,
    },
  );
}

function toScanSummary(workspace: Workspace): ScanSummary {
  return {
    scan_id: workspace.flow.id,
    status: workspace.flow.status as ScanSummary["status"],
    target: workspace.flow.target,
    started_at: workspace.flow.created_at,
    findings_summary: summarizeFindings(workspace.findings),
    current_step: workspace.actions.length,
    total_steps: Math.max(
      workspace.tasks.length + workspace.subtasks.length,
      workspace.actions.length,
    ),
    active_stage:
      workspace.tasks.find((task) => task.status === "running")?.name ?? null,
    pending_reason: workspace.needs_review ? "approval required" : null,
  };
}

function toScanSteps(workspace: Workspace): ScanStep[] {
  const taskSteps = workspace.tasks.map((task, index) => ({
    step_id: index + 1,
    step_number: index + 1,
    phase: (index === 0
      ? "recon"
      : index === workspace.tasks.length - 1
        ? "report"
        : "scan") as ScanStep["phase"],
    decision: task.name,
    action: task.agent_role,
    summary: task.description,
    created_at: task.created_at,
  }));

  const actionSteps = workspace.actions.map((action, index) => ({
    step_id: taskSteps.length + index + 1,
    step_number: taskSteps.length + index + 1,
    phase: "scan" as const,
    decision: action.function_name,
    action: action.execution_mode || action.agent_role,
    summary:
      typeof action.output?.summary === "string"
        ? String(action.output.summary)
        : JSON.stringify(action.output),
    created_at: action.updated_at,
  }));

  return [...taskSteps, ...actionSteps];
}

function toAgentGraph(workspace: Workspace): AgentGraph {
  return {
    nodes: workspace.agents.map((agent) => ({
      agent_id: agent.id,
      name: agent.role,
      role: agent.role,
      status: agent.status,
      created_at: agent.created_at,
    })),
    edges: [],
  };
}

function toToolEvents(workspace: Workspace): ToolStreamEvent[] {
  return workspace.actions.map((action, index) => ({
    tool_call_id: index + 1,
    tool_name: action.function_name,
    status: action.status,
    phase: "scan",
    started_at: action.created_at,
    finished_at: action.updated_at,
  }));
}

function toScanReport(workspace: Workspace): ScanReport {
  const findings: ScanFinding[] = workspace.findings.map((finding) => ({
    title: finding.title,
    description: finding.description,
    severity: finding.severity as ScanFinding["severity"],
    confirmed: true,
    verification_status: "confirmed",
  }));

  return {
    scan_id: workspace.flow.id,
    summary: workspace.flow.objective,
    findings,
    report_version: "go-v2",
    pdf_status: "ready",
  };
}

export async function listScans() {
  const payload = await listFlows();
  const batch = await listWorkspaces(payload.flows.map((flow) => flow.id));
  const workspaces = batch.workspaces ?? [];

  for (const [flowId, message] of Object.entries(batch.errors ?? {})) {
    console.error(`Failed to load workspace for flow ${flowId}: ${message}`);
  }

  if (workspaces.length === 0 && payload.flows.length > 0) {
    throw new Error(
      `Failed to load workspaces for all ${payload.flows.length} flow(s)`,
    );
  }

  const summaries = new Map(
    workspaces.map((workspace) => [workspace.flow.id, toScanSummary(workspace)]),
  );
  return payload.flows
    .map((flow) => summaries.get(flow.id))
    .filter((summary): summary is ScanSummary => summary !== undefined);
}

export async function getScanStatus(scanId: string) {
  return toScanSummary(await getWorkspace(scanId));
}

export async function getScanSteps(scanId: string) {
  return toScanSteps(await getWorkspace(scanId));
}

export async function getScanGraph(scanId: string) {
  return toAgentGraph(await getWorkspace(scanId));
}

export async function getScanReport(scanId: string) {
  return toScanReport(await getWorkspace(scanId));
}

export async function generateScanReportPdf(scanId: string) {
  await downloadReport(scanId, "pdf");
  return toScanReport(await getWorkspace(scanId));
}

export async function cancelScan(scanId: string) {
  await apiFetch<{ status: string; flow_id: string }>(`/v1/flows/${scanId}/cancel`, {
    method: "POST",
  });
  return getScanStatus(scanId);
}

export async function startScan(payload: {
  target_url: string;
  scope?: "unauthenticated" | "authenticated";
  mode?: "quick" | "standard" | "deep" | "fast" | "normal";
  notes?: string;
}) {
  const flow = await createFlow({
    name: payload.target_url,
    target: payload.target_url,
    objective:
      payload.notes?.trim() ||
      "Autonomous assessment requested from frontend workflow.",
  });
  await startFlow(flow.id);
  return {
    scan_id: flow.id,
    status: "queued",
  };
}

export async function getScanAssist(payload: {
  target_url: string;
  scope?: "unauthenticated" | "authenticated";
  mode?: "quick" | "standard" | "deep" | "fast" | "normal";
  notes?: string;
}) {
  return {
    summary: `Target ${payload.target_url} will be queued as a flow in the Go control plane.`,
    suggested_scope: payload.scope || "unauthenticated",
    suggested_mode: payload.mode || "standard",
    guardrails: [
      "Stay within authorized scope.",
      "Review approvals before risky execution.",
      "Capture evidence and findings in the workspace.",
    ],
  } satisfies ScanAssistResponse;
}

export async function getReportSummary(scanId: string) {
  const workspace = await getWorkspace(scanId);
  return {
    summary: workspace.flow.objective,
    remediation: workspace.findings.map((finding) => finding.title),
  } satisfies ReportSummaryResponse;
}

export async function getLiveChat(payload: {
  scan_id: string;
  question: string;
}) {
  const workspace = await getWorkspace(payload.scan_id);
  const question = payload.question.trim().toLowerCase();
  if (!question) {
    return {
      answer: "Ask about findings, approvals, or recent actions.",
    } satisfies LiveChatResponse;
  }

  if (question.includes("approval")) {
    const pending = workspace.approvals.filter(
      (approval) => approval.status === "pending",
    );
    return {
      answer:
        pending.length > 0
          ? `${pending.length} approval${pending.length === 1 ? "" : "s"} still need review.`
          : "There are no pending approvals on this flow.",
    } satisfies LiveChatResponse;
  }

  if (question.includes("finding")) {
    if (workspace.findings.length === 0) {
      return {
        answer: "No findings have been recorded yet.",
      } satisfies LiveChatResponse;
    }
    const titles = workspace.findings
      .slice(0, 3)
      .map((finding) => finding.title)
      .join(", ");
    return {
      answer: `Current findings: ${titles}.`,
    } satisfies LiveChatResponse;
  }

  const latestAction = workspace.actions.at(-1);
  return {
    answer: latestAction
      ? `Latest action: ${latestAction.function_name} is ${latestAction.status}.`
      : `Flow ${payload.scan_id} has not produced any actions yet.`,
  } satisfies LiveChatResponse;
}

export async function reviewApproval(
  approvalId: string,
  approved: boolean,
  note: string,
) {
  return apiFetch<Approval>(`/v1/approvals/${approvalId}/review`, {
    method: "POST",
    body: JSON.stringify({ approved, note }),
  });
}

export async function downloadReport(
  flowId: string,
  format: "markdown" | "json" | "pdf",
) {
  const response = await fetch(
    `${API_BASE}/v1/flows/${flowId}/report?format=${format}`,
    {
      cache: "no-store",
    },
  );
  if (!response.ok) {
    throw new Error(`report download failed with ${response.status}`);
  }
  const blob = await response.blob();
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = `${flowId}-report.${format === "markdown" ? "md" : format}`;
  try {
    document.body.appendChild(anchor);
    anchor.click();
  } finally {
    anchor.remove();
    URL.revokeObjectURL(url);
  }
}

export async function streamFlowEvents(
  flowId: string,
  signal: AbortSignal,
  onEvent: (event: FlowEvent) => void,
): Promise<void> {
  const response = await fetch(`${API_BASE}/v1/flows/${flowId}/events`, {
    method: "GET",
    headers: { Accept: "text/event-stream" },
    signal,
    cache: "no-store",
  });

  if (!response.ok) {
    throw new Error(`event stream failed with ${response.status}`);
  }
  if (!response.body) {
    throw new Error("event stream is unavailable in this browser");
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  while (!signal.aborted) {
    const { value, done } = await reader.read();
    if (done) {
      break;
    }
    buffer += decoder.decode(value, { stream: true });

    let blockEnd = buffer.indexOf("\n\n");
    while (blockEnd >= 0) {
      const block = buffer.slice(0, blockEnd);
      buffer = buffer.slice(blockEnd + 2);
      blockEnd = buffer.indexOf("\n\n");

      const lines = block.split(/\r?\n/);
      let type = "message";
      const data: string[] = [];

      for (const line of lines) {
        if (line.startsWith(":") || line.trim() === "") {
          continue;
        }
        if (line.startsWith("event:")) {
          type = line.slice("event:".length).trim();
          continue;
        }
        if (line.startsWith("data:")) {
          data.push(line.slice("data:".length).trim());
        }
      }

      if (data.length === 0) {
        continue;
      }

      try {
        const payload = JSON.parse(data.join("\n")) as FlowEvent;
        onEvent({ ...payload, type });
      } catch {
        onEvent({ type, message: data.join("\n") });
      }
    }
  }
}
