"use client";

import { useEffect, useRef, useState } from "react";
import { getWorkspace, streamFlowEvents } from "./api";
import type {
  AgentGraph,
  FlowEvent,
  ScanStep,
  ScanSummary,
  ToolStreamEvent,
  Workspace,
} from "./types";

export interface FlowStreamState {
  workspace: Workspace | null;
  events: FlowEvent[];
  loading: boolean;
  error: string | null;
  connected: boolean;
}

export interface ScanStreamState {
  status: ScanSummary | null;
  steps: ScanStep[];
  agentGraph: AgentGraph | null;
  toolEvents: ToolStreamEvent[];
  loading: boolean;
  error: string | null;
  done: boolean;
  connected: boolean;
}

export interface ReconnectOptions {
  reconnect?: boolean;
  reconnectDelay?: number;
  maxReconnects?: number;
}

type FlowStreamSnapshot = {
  flowId: string | null;
  workspace: Workspace | null;
  events: FlowEvent[];
  loading: boolean;
  error: string | null;
  connected: boolean;
};

function summarizeFindings(workspace: Workspace) {
  return workspace.findings.reduce(
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
    findings_summary: summarizeFindings(workspace),
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
  return workspace.actions.map((action, index) => ({
    step_id: index + 1,
    step_number: index + 1,
    phase: "scan",
    decision: action.function_name,
    action: action.agent_role,
    summary:
      typeof action.output?.summary === "string"
        ? String(action.output.summary)
        : JSON.stringify(action.output),
    created_at: action.updated_at,
  }));
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

export function useFlowStream(
  flowId: string | null,
  options?: ReconnectOptions,
): FlowStreamState {
  const [state, setState] = useState<FlowStreamSnapshot>({
    flowId,
    workspace: null,
    events: [],
    loading: Boolean(flowId),
    error: null,
    connected: false,
  });
  const eventIds = useRef<Set<string>>(new Set());
  const optionsRef = useRef(options);
  optionsRef.current = options;

  useEffect(() => {
    if (!flowId) {
      eventIds.current.clear();
      return;
    }

    const currentFlowId = flowId;
    let active = true;
    let retryCount = 0;
    let retryTimer: ReturnType<typeof setTimeout> | null = null;
    let controller = new AbortController();

    async function loadWorkspace(fid: string) {
      const next = await getWorkspace(fid);
      if (active) {
        setState((current) => ({
          flowId: fid,
          workspace: next,
          events: current.flowId == fid ? current.events : [],
          loading: false,
          error: null,
          connected: current.flowId == fid ? current.connected : false,
        }));
      }
    }

    function scheduleReconnect(streamError?: unknown) {
      if (retryTimer) clearTimeout(retryTimer);
      if (!active) return;

      const reconnect = optionsRef.current?.reconnect ?? true;
      const initialDelay = optionsRef.current?.reconnectDelay ?? 1000;
      const maxRetries = optionsRef.current?.maxReconnects ?? 10;

      if (reconnect && retryCount < maxRetries) {
        const delay = Math.min(
          initialDelay * Math.pow(2, retryCount),
          30000,
        );
        retryCount++;
        setState((current) => ({
          flowId: currentFlowId,
          workspace:
            current.flowId == currentFlowId ? current.workspace : null,
          events:
            current.flowId == currentFlowId ? current.events : [],
          loading: false,
          error: null,
          connected: false,
        }));
        retryTimer = setTimeout(() => {
          if (active) connect();
        }, delay);
      } else {
        setState((current) => ({
          flowId: currentFlowId,
          workspace:
            current.flowId == currentFlowId ? current.workspace : null,
          events:
            current.flowId == currentFlowId ? current.events : [],
          loading: false,
          error:
            streamError instanceof Error
              ? streamError.message
              : "event stream disconnected",
          connected: false,
        }));
      }
    }

    function connect() {
      controller.abort();
      controller = new AbortController();

      void loadWorkspace(currentFlowId).catch((err) => {
        if (active) {
          setState({
            flowId: currentFlowId,
            workspace: null,
            events: [],
            loading: false,
            error: err instanceof Error ? err.message : "load failed",
            connected: false,
          });
        }
      });

      void streamFlowEvents(
        currentFlowId,
        controller.signal,
        (event) => {
          retryCount = 0;
          if (event.type === "snapshot") {
            void loadWorkspace(currentFlowId).catch((wsErr) => {
              if (active) {
                console.warn("[sse] workspace sync failed on snapshot:", wsErr);
              }
            });
            return;
          }
          if (event.id && eventIds.current.has(event.id)) {
            return;
          }
          if (event.id) {
            eventIds.current.add(event.id);
          }
          setState((current) => ({
            flowId: currentFlowId,
            workspace:
              current.flowId == currentFlowId ? current.workspace : null,
            events:
              current.flowId == currentFlowId
                ? [event, ...current.events].slice(0, 80)
                : [event],
            loading: false,
            error: null,
            connected: true,
          }));
          if (
            event.type.startsWith("flow.") ||
            event.type.startsWith("approval.") ||
            event.type.startsWith("action.")
          ) {
            void loadWorkspace(currentFlowId).catch((wsErr) => {
              if (active) {
                console.warn("[sse] workspace sync failed on event:", wsErr);
              }
            });
          }
        },
      )
        .then(() => {
          if (!controller.signal.aborted && active) {
            scheduleReconnect();
          }
        })
        .catch((streamError) => {
          if (!controller.signal.aborted && active) {
            scheduleReconnect(streamError);
          }
        });
    }

    eventIds.current.clear();
    connect();

    return () => {
      active = false;
      controller.abort();
      if (retryTimer) clearTimeout(retryTimer);
    };
  }, [flowId]);

  if (!flowId) {
    return {
      workspace: null,
      events: [],
      loading: false,
      error: null,
      connected: false,
    };
  }

  if (state.flowId !== flowId) {
    return {
      workspace: null,
      events: [],
      loading: true,
      error: null,
      connected: false,
    };
  }

  return {
    workspace: state.workspace,
    events: state.events,
    loading: state.loading,
    error: state.error,
    connected: state.connected,
  };
}

export function useScanStream(
  scanId: string | null,
  enabled?: boolean,
  options?: ReconnectOptions,
): ScanStreamState {
  const effectiveId = (enabled ?? true) ? scanId : null;
  const state = useFlowStream(effectiveId, options);
  return {
    status: state.workspace ? toScanSummary(state.workspace) : null,
    steps: state.workspace ? toScanSteps(state.workspace) : [],
    agentGraph: state.workspace ? toAgentGraph(state.workspace) : null,
    toolEvents: state.workspace ? toToolEvents(state.workspace) : [],
    loading: state.loading,
    error: state.error,
    done: Boolean(
      state.workspace &&
      (state.workspace.flow.status === "completed" ||
        state.workspace.flow.status === "failed"),
    ),
    connected: state.connected,
  };
}
