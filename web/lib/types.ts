export type Status =
  | "pending"
  | "queued"
  | "running"
  | "completed"
  | "failed"
  | "cancelled";

export interface Flow {
  id: string;
  tenant_id: string;
  name: string;
  target: string;
  objective: string;
  status: Status;
  created_at: string;
  updated_at: string;
}

export interface Task {
  id: string;
  flow_id: string;
  name: string;
  description: string;
  status: Status;
  agent_role: string;
  created_at: string;
  updated_at: string;
}

export interface Subtask {
  id: string;
  task_id: string;
  flow_id: string;
  name: string;
  description: string;
  status: Status;
  agent_role: string;
  created_at: string;
  updated_at: string;
}

export interface Action {
  id: string;
  flow_id: string;
  task_id: string;
  subtask_id: string;
  agent_role: string;
  function_name: string;
  input: Record<string, unknown>;
  output: Record<string, unknown>;
  status: Status;
  execution_mode: string;
  created_at: string;
  updated_at: string;
}

export interface Artifact {
  id: string;
  flow_id: string;
  action_id: string;
  kind: string;
  name: string;
  content: string;
  metadata: Record<string, unknown>;
  created_at: string;
}

export interface Memory {
  id: string;
  flow_id: string;
  action_id: string;
  kind: string;
  content: string;
  metadata: Record<string, unknown>;
  created_at: string;
}

export interface Finding {
  id: string;
  flow_id: string;
  title: string;
  severity: "info" | "low" | "medium" | "high" | "critical" | string;
  description: string;
  created_at: string;
}

export interface Agent {
  id: string;
  flow_id: string;
  role: string;
  model: string;
  status: Status;
  created_at: string;
  updated_at: string;
}

export interface Execution {
  id: string;
  flow_id: string;
  action_id: string;
  profile: string;
  runtime: string;
  metadata: Record<string, unknown>;
  status: Status;
  started_at: string;
  completed_at?: string;
}

export interface Approval {
  id: string;
  flow_id: string;
  tenant_id: string;
  kind: string;
  status: string;
  requested_by: string;
  reviewed_by: string;
  review_note: string;
  reason: string;
  payload: Record<string, unknown>;
  created_at: string;
  reviewed_at?: string;
}

export interface FunctionDef {
  name: string;
  description: string;
  profile: string;
  category?: string;
  safety_profile?: string;
  requires_network?: boolean;
  requires_pentest_image?: boolean;
  input_schema?: {
    name: string;
    type: string;
    description: string;
    required: boolean;
  }[];
}

export interface Workspace {
  flow: Flow;
  tasks: Task[];
  subtasks: Subtask[];
  actions: Action[];
  artifacts: Artifact[];
  memories: Memory[];
  findings: Finding[];
  agents: Agent[];
  executions: Execution[];
  approvals: Approval[];
  functions: FunctionDef[];
  tenant_id: string;
  queue_mode: string;
  executor_mode: string;
  browser_mode: string;
  executor_network_mode: string;
  executor_network_name: string;
  executor_net_raw_enabled: boolean;
  terminal_network_enabled: boolean;
  risky_approval_required: boolean;
  flow_max_concurrent_actions: number;
  flow_min_action_interval_ms: number;
  browser_warning: string;
  network_warning: string;
  risk_warning: string;
  needs_review: boolean;
}

export interface FlowEvent {
  id?: string;
  sequence?: number;
  flow_id?: string;
  type: string;
  message?: string;
  payload?: Record<string, unknown>;
  created_at?: string;
}

export interface ArchitectureResponse {
  name: string;
  services: string[];
  model: string[];
  transport: Record<string, string>;
  executor_mode: string;
  browser_mode: string;
  executor_network_mode: string;
  executor_network_name: string;
  executor_net_raw_enabled: boolean;
  terminal_network_enabled: boolean;
  risky_approval_required: boolean;
  flow_max_concurrent_actions: number;
  flow_min_action_interval_ms: number;
  browser_warning: string;
  network_warning: string;
  risk_warning: string;
  version: string;
}

export interface FindingsSummary {
  critical: number;
  high: number;
  medium: number;
  low: number;
  info: number;
}

export type ScanStatus =
  | "queued"
  | "running"
  | "completed"
  | "failed"
  | "cancelled"
  | "canceled";

export type Phase = "recon" | "scan" | "validate" | "report";

export interface ScanStatusData {
  scan_id: string;
  status: ScanStatus;
  mode?: string | null;
  target?: string | null;
  current_step?: number;
  total_steps?: number;
  active_stage?: string | null;
  pending_reason?: string | null;
}

export interface ScanSummary extends ScanStatusData {
  started_at?: string;
  findings_summary?: FindingsSummary | null;
}

export interface ScanStep {
  step_id: number;
  step_number?: number | null;
  phase: Phase;
  decision: string;
  action?: string;
  summary?: string;
  created_at?: string;
}

export interface ToolStreamEvent {
  tool_call_id: number;
  tool_name: string;
  status: "running" | "completed" | "failed" | string;
  phase?: Phase | string;
  error?: string | null;
  started_at?: string | null;
  finished_at?: string | null;
  duration_seconds?: number | null;
}

export interface ScanFinding {
  title?: string;
  description?: string;
  severity?: "info" | "low" | "medium" | "high" | "critical";
  confirmed?: boolean | null;
  verification_status?: "confirmed" | "unconfirmed" | string;
  candidate?: boolean | null;
  confidence?: number | null;
  affected_component?: string | null;
  impact?: string | null;
  evidence?: string | null;
  request?: string | null;
  response?: string | null;
  steps_to_reproduce?: string | null;
  recommendation?: string | null;
  solution?: string | null;
  cvss_score?: number | null;
  cvss_vector?: string | null;
  cve_ids?: string[] | null;
  references?: string[] | null;
  matched_at?: string;
  source?: string;
  [key: string]: unknown;
}

export interface ScanReport {
  scan_id: string;
  summary: string;
  findings: ScanFinding[];
  report_version?: string;
  phase_metrics?: Record<
    string,
    {
      total: number;
      success: number;
      failed: number;
      timeout: number;
      avg_duration: number;
    }
  > | null;
  timed_out_tools?:
    | {
        tool_name: string;
        duration_seconds?: number | null;
        error?: string | null;
      }[]
    | null;
  pdf_status?: "pending" | "ready" | "failed";
  pdf_size_bytes?: number | null;
  pdf_generated_at?: string | null;
  pdf_url?: string | null;
}

export interface ScanAssistResponse {
  summary: string;
  suggested_scope: string;
  suggested_mode: string;
  guardrails: string[];
}

export interface ReportSummaryResponse {
  summary: string;
  remediation: string[];
}

export interface LiveChatResponse {
  answer: string;
}

export interface AgentGraphNode {
  agent_id: string;
  parent_id?: string | null;
  name: string;
  role: string;
  status: string;
  skills?: string[] | null;
  created_at?: string | null;
}

export interface AgentGraphEdge {
  from_agent_id: string;
  to_agent_id: string;
  edge_type: string;
  created_at?: string | null;
}

export interface AgentGraph {
  nodes: AgentGraphNode[];
  edges: AgentGraphEdge[];
}
