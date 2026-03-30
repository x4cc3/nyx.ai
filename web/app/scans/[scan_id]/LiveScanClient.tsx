"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import AppShell from "@/components/layout/AppShell";
import { Badge, Button, Card, CardContent, ConfirmDialog, Skeleton } from "@/components/ui";
import { cancelScan, reviewApproval, startFlow } from "@/lib/api";
import { useFlowStream } from "@/lib/sse";
import type {
  AgentGraph,
  FlowEvent,
  ScanStatusData,
  Workspace,
} from "@/lib/types";
import {
  Activity,
  AlertCircle,
  ArrowRight,
  Ban,
  CheckCircle2,
  Clock3,
  FileBox,
  Play,
  RefreshCw,
  ShieldCheck,
  TerminalSquare,
} from "lucide-react";

function formatTime(value?: string) {
  if (!value) return "unknown";
  try {
    return new Intl.DateTimeFormat(undefined, {
      month: "short",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
    }).format(new Date(value));
  } catch {
    return value;
  }
}

function formatPayload(value: unknown) {
  if (value == null) return "no payload";
  if (typeof value === "string") return value;
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}

function metadataValue(
  metadata: Record<string, unknown> | undefined,
  key: string,
): string | null {
  const value = metadata?.[key];
  if (typeof value !== "string") return null;
  const trimmed = value.trim();
  return trimmed.length ? trimmed : null;
}

function truncate(value: string, limit = 120) {
  if (value.length <= limit) return value;
  return `${value.slice(0, limit - 1)}…`;
}

function statusVariant(status?: string) {
  switch (status) {
    case "running":
      return "accent" as const;
    case "completed":
      return "success" as const;
    case "failed":
      return "hot" as const;
    case "cancelled":
    case "queued":
      return "warn" as const;
    default:
      return "outline" as const;
  }
}

function toScanStatus(workspace: Workspace | null): ScanStatusData | null {
  if (!workspace) return null;
  return {
    scan_id: workspace.flow.id,
    status: workspace.flow.status as ScanStatusData["status"],
    target: workspace.flow.target,
    current_step: workspace.actions.length,
    total_steps: Math.max(
      workspace.actions.length,
      workspace.tasks.length + workspace.subtasks.length,
    ),
    active_stage:
      workspace.tasks.find((task) => task.status === "running")?.name ?? null,
    pending_reason: workspace.needs_review ? "approval required" : null,
  };
}

function toAgentGraph(workspace: Workspace | null): AgentGraph | null {
  if (!workspace) return null;
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

function EventRow({ event }: { event: FlowEvent }) {
  return (
    <div className="grid gap-2 border-b border-border px-4 py-3 md:grid-cols-[140px,minmax(0,1fr),170px]">
      <div className="text-xs font-medium text-foreground">{event.type}</div>
      <div className="min-w-0 text-sm leading-6 text-muted-foreground">
        {event.message || formatPayload(event.payload)}
      </div>
      <div className="text-xs text-muted-foreground">
        {formatTime(event.created_at)}
      </div>
    </div>
  );
}

export default function LiveScanClient({ scanId }: { scanId: string }) {
  const { workspace, events, loading, error, connected } = useFlowStream(scanId);
  const [actionError, setActionError] = useState<string | null>(null);
  const [busyAction, setBusyAction] = useState<string | null>(null);
  const [confirmCancel, setConfirmCancel] = useState(false);
  const [confirmRejectId, setConfirmRejectId] = useState<string | null>(null);

  const scan = useMemo(() => toScanStatus(workspace), [workspace]);
  const agentGraph = useMemo(() => toAgentGraph(workspace), [workspace]);

  async function handleStartFlow() {
    setBusyAction("start");
    setActionError(null);
    try {
      await startFlow(scanId);
    } catch (startError) {
      setActionError(
        startError instanceof Error ? startError.message : "start failed",
      );
    } finally {
      setBusyAction(null);
    }
  }

  async function handleReview(approvalId: string, approved: boolean) {
    setBusyAction(approvalId);
    setActionError(null);
    try {
      await reviewApproval(
        approvalId,
        approved,
        approved ? "approved from workspace" : "rejected from workspace",
      );
    } catch (reviewError) {
      setActionError(
        reviewError instanceof Error ? reviewError.message : "review failed",
      );
    } finally {
      setBusyAction(null);
    }
  }

  async function handleCancelFlow() {
    setConfirmCancel(false);
    setBusyAction("cancel");
    setActionError(null);
    try {
      await cancelScan(scanId);
    } catch (cancelError) {
      setActionError(
        cancelError instanceof Error ? cancelError.message : "cancel failed",
      );
    } finally {
      setBusyAction(null);
    }
  }

  return (
    <AppShell scan={scan} agentGraph={agentGraph} showContext>
      <div className="space-y-6 px-4 py-4 md:px-6 md:py-6 xl:px-8">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div className="space-y-2">
            <div className="flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
              <span>Flow</span>
              <span className={connected ? "text-nyx-success" : "text-nyx-warn"}>
                {connected ? "Connected" : "Connection degraded"}
              </span>
            </div>
            <h1 className="text-2xl font-semibold text-foreground">
              {workspace?.flow.name ?? "Loading flow"}
            </h1>
            <div className="flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
              <Badge variant={statusVariant(workspace?.flow.status)}>
                {workspace?.flow.status ?? "loading"}
              </Badge>
              {workspace?.queue_mode ? (
                <Badge variant="outline">queue {workspace.queue_mode}</Badge>
              ) : null}
              {workspace?.executor_mode ? (
                <Badge variant="outline">exec {workspace.executor_mode}</Badge>
              ) : null}
              {workspace?.browser_mode ? (
                <Badge
                  variant={
                    workspace.browser_mode === "http" ? "warn" : "outline"
                  }
                >
                  browser {workspace.browser_mode}
                </Badge>
              ) : null}
              {workspace?.terminal_network_enabled ? (
                <Badge variant="warn">
                  net{" "}
                  {workspace.executor_network_mode === "custom" &&
                  workspace.executor_network_name
                    ? workspace.executor_network_name
                    : workspace.executor_network_mode}
                </Badge>
              ) : (
                <Badge variant="outline">net isolated</Badge>
              )}
              {workspace?.executor_net_raw_enabled ? (
                <Badge variant="warn">NET_RAW available</Badge>
              ) : null}
              {workspace?.tenant_id ? (
                <Badge variant="outline">tenant {workspace.tenant_id}</Badge>
              ) : null}
              <span className="font-mono">{workspace?.flow.target ?? scanId}</span>
            </div>
            {workspace?.flow.objective ? (
              <p className="max-w-3xl text-sm leading-6 text-muted-foreground">
                {workspace.flow.objective}
              </p>
            ) : null}
            {workspace?.network_warning ? (
              <div className="flex max-w-3xl items-start gap-2 rounded-md border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-sm text-amber-100">
                <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
                <p>{workspace.network_warning}</p>
              </div>
            ) : null}
            {workspace?.browser_warning ? (
              <div className="flex max-w-3xl items-start gap-2 rounded-md border border-sky-500/30 bg-sky-500/10 px-3 py-2 text-sm text-sky-100">
                <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
                <p>{workspace.browser_warning}</p>
              </div>
            ) : null}
            {workspace?.risk_warning ? (
              <div className="flex max-w-3xl items-start gap-2 rounded-md border border-border bg-background px-3 py-2 text-sm text-muted-foreground">
                <ShieldCheck className="mt-0.5 h-4 w-4 shrink-0 text-foreground" />
                <p>{workspace.risk_warning}</p>
              </div>
            ) : null}
          </div>

          <div className="flex flex-wrap gap-2">
            <Button asChild variant="secondary" size="sm">
              <Link href="/dashboard">Back</Link>
            </Button>
            <Button asChild variant="secondary" size="sm">
              <Link href={`/scans/${scanId}/report`}>Report</Link>
            </Button>
            <Button
              size="sm"
              onClick={handleStartFlow}
              disabled={
                busyAction === "start" ||
                workspace?.flow.status === "cancelled" ||
                workspace?.flow.status === "completed" ||
                workspace?.flow.status === "failed"
              }
            >
              {busyAction === "start" ? (
                <RefreshCw className="mr-1.5 h-4 w-4 animate-spin" />
              ) : (
                <Play className="mr-1.5 h-4 w-4" />
              )}
              Start flow
            </Button>
            <Button
              size="sm"
              variant="secondary"
              onClick={() => setConfirmCancel(true)}
              disabled={
                busyAction === "cancel" ||
                workspace?.flow.status === "cancelled" ||
                workspace?.flow.status === "completed" ||
                workspace?.flow.status === "failed"
              }
            >
              {busyAction === "cancel" ? (
                <RefreshCw className="mr-1.5 h-4 w-4 animate-spin" />
              ) : (
                <Ban className="mr-1.5 h-4 w-4" />
              )}
              Cancel flow
            </Button>
          </div>
        </div>

        {error || actionError ? (
          <div className="rounded-md border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-200">
            {actionError || error}
          </div>
        ) : null}

        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
          {[
            {
              label: "Actions",
              value: workspace?.actions.length ?? 0,
              hint: "Function executions",
            },
            {
              label: "Findings",
              value: workspace?.findings.length ?? 0,
              hint: "Captured issues",
            },
            {
              label: "Artifacts",
              value: workspace?.artifacts.length ?? 0,
              hint: "Evidence objects",
            },
            {
              label: "Approvals",
              value:
                workspace?.approvals.filter((approval) => approval.status === "pending")
                  .length ?? 0,
              hint: "Waiting review",
            },
          ].map((item) => (
            <Card key={item.label} className="border-border bg-card">
              <CardContent className="space-y-2 p-4">
                <p className="text-sm font-medium text-foreground">{item.label}</p>
                <p className="text-2xl font-semibold text-foreground">{item.value}</p>
                <p className="text-sm text-muted-foreground">{item.hint}</p>
              </CardContent>
            </Card>
          ))}
        </div>

        <div className="grid gap-4 xl:grid-cols-[1.35fr,0.9fr]">
          <div className="space-y-4">
            <Card className="border-border bg-card">
              <CardContent className="p-0">
                <div className="border-b border-border px-4 py-3">
                  <h2 className="text-base font-semibold text-foreground">
                    Execution log
                  </h2>
                  <p className="mt-1 text-sm text-muted-foreground">
                    Tasks, subtasks, and action outputs mapped to the workspace.
                  </p>
                </div>

                {loading && !workspace ? (
                  <div className="space-y-3 p-4">
                    <Skeleton className="h-24 w-full rounded-md" />
                    <Skeleton className="h-24 w-full rounded-md" />
                  </div>
                ) : workspace && workspace.actions.length > 0 ? (
                  <div className="divide-y divide-border">
                    {workspace.actions
                      .slice()
                      .reverse()
                      .map((action) => (
                        <div key={action.id} className="space-y-3 px-4 py-4">
                          <div className="flex flex-col gap-2 md:flex-row md:items-start md:justify-between">
                            <div className="space-y-1">
                              <div className="flex flex-wrap items-center gap-2">
                                <Badge variant={statusVariant(action.status)}>
                                  {action.status}
                                </Badge>
                                <span className="text-sm font-semibold text-foreground">
                                  {action.function_name}
                                </span>
                                <span className="text-xs font-mono text-muted-foreground">
                                  {action.agent_role}
                                </span>
                              </div>
                              <div className="text-xs text-muted-foreground">
                                {action.execution_mode} / updated {formatTime(action.updated_at)}
                              </div>
                            </div>
                            <div className="text-xs font-mono text-muted-foreground">
                              {action.id.slice(0, 8)}
                            </div>
                          </div>

                          <div className="grid gap-3 lg:grid-cols-2">
                            <div className="rounded-md border border-border bg-background px-3 py-3">
                              <div className="mb-2 text-xs font-medium text-muted-foreground">
                                Input
                              </div>
                              <pre className="overflow-x-auto whitespace-pre-wrap break-all text-xs leading-6 text-foreground/80">
                                {formatPayload(action.input)}
                              </pre>
                            </div>
                            <div className="rounded-md border border-border bg-background px-3 py-3">
                              <div className="mb-2 text-xs font-medium text-muted-foreground">
                                Output
                              </div>
                              <pre className="overflow-x-auto whitespace-pre-wrap break-all text-xs leading-6 text-foreground/80">
                                {formatPayload(action.output)}
                              </pre>
                            </div>
                          </div>
                        </div>
                      ))}
                  </div>
                ) : (
                  <div className="px-4 py-12 text-center">
                    <TerminalSquare className="mx-auto h-5 w-5 text-muted-foreground" />
                    <p className="mt-3 text-sm text-muted-foreground">
                      No actions yet.
                    </p>
                  </div>
                )}
              </CardContent>
            </Card>

            <Card className="border-border bg-card">
              <CardContent className="p-0">
                <div className="border-b border-border px-4 py-3">
                  <h2 className="text-base font-semibold text-foreground">
                    Event stream
                  </h2>
                  <p className="mt-1 text-sm text-muted-foreground">
                    Persisted flow events from the SSE stream.
                  </p>
                </div>
                {events.length > 0 ? (
                  <div className="divide-y divide-border">
                    {events.slice(0, 24).map((event, index) => (
                      <EventRow
                        key={event.id || `${event.type}-${event.created_at || index}`}
                        event={event}
                      />
                    ))}
                  </div>
                ) : (
                  <div className="px-4 py-12 text-center">
                    <Activity className="mx-auto h-5 w-5 text-muted-foreground" />
                    <p className="mt-3 text-sm text-muted-foreground">
                      Waiting for runtime events.
                    </p>
                  </div>
                )}
              </CardContent>
            </Card>
          </div>

          <div className="space-y-4">
            <Card className="border-border bg-card">
              <CardContent className="space-y-4 p-4">
                <div className="flex items-center gap-2">
                  <ShieldCheck className="h-4 w-4 text-muted-foreground" />
                  <h2 className="text-base font-semibold text-foreground">
                    Approvals
                  </h2>
                </div>
                <div className="space-y-3">
                  {workspace?.approvals.length ? (
                    workspace.approvals.map((approval) => (
                      <div
                        key={approval.id}
                        className="rounded-md border border-border bg-background px-3 py-3"
                      >
                        <div className="flex items-center justify-between gap-3">
                          <div className="text-sm font-semibold text-foreground">
                            {approval.kind}
                          </div>
                          <Badge variant={statusVariant(approval.status)}>
                            {approval.status}
                          </Badge>
                        </div>
                        <div className="mt-2 text-sm leading-6 text-muted-foreground">
                          {approval.reason}
                        </div>
                        <div className="mt-2 text-xs text-muted-foreground">
                          requested {formatTime(approval.created_at)}
                        </div>
                        {approval.status === "pending" ? (
                          <div className="mt-3 flex gap-2">
                            <Button
                              size="sm"
                              onClick={() => handleReview(approval.id, true)}
                              disabled={busyAction === approval.id}
                            >
                              Approve
                            </Button>
                            <Button
                              size="sm"
                              variant="secondary"
                              onClick={() => setConfirmRejectId(approval.id)}
                              disabled={busyAction === approval.id}
                            >
                              Reject
                            </Button>
                          </div>
                        ) : null}
                      </div>
                    ))
                  ) : (
                    <p className="text-sm text-muted-foreground">
                      No approvals requested.
                    </p>
                  )}
                </div>
              </CardContent>
            </Card>

            <Card className="border-border bg-card">
              <CardContent className="space-y-4 p-4">
                <div className="flex items-center gap-2">
                  <AlertCircle className="h-4 w-4 text-muted-foreground" />
                  <h2 className="text-base font-semibold text-foreground">
                    Findings
                  </h2>
                </div>
                <div className="space-y-3">
                  {workspace?.findings.length ? (
                    workspace.findings.map((finding) => (
                      <div
                        key={finding.id}
                        className="rounded-md border border-border bg-background px-3 py-3"
                      >
                        <div className="flex items-center justify-between gap-3">
                          <div className="text-sm font-semibold text-foreground">
                            {finding.title}
                          </div>
                          <Badge variant={statusVariant(finding.severity)}>
                            {finding.severity}
                          </Badge>
                        </div>
                        <div className="mt-2 text-sm leading-6 text-muted-foreground">
                          {finding.description}
                        </div>
                      </div>
                    ))
                  ) : (
                    <p className="text-sm text-muted-foreground">
                      No findings captured yet.
                    </p>
                  )}
                </div>
              </CardContent>
            </Card>

            <Card className="border-border bg-card">
              <CardContent className="space-y-4 p-4">
                <div className="flex items-center gap-2">
                  <FileBox className="h-4 w-4 text-muted-foreground" />
                  <h2 className="text-base font-semibold text-foreground">
                    Executions
                  </h2>
                </div>
                <div className="space-y-3">
                  {workspace?.executions.length ? (
                    workspace.executions
                      .slice()
                      .reverse()
                      .slice(0, 8)
                      .map((execution) => {
                        const image = metadataValue(execution.metadata, "image");
                        const networkMode = metadataValue(
                          execution.metadata,
                          "network_mode",
                        );
                        const networkName = metadataValue(
                          execution.metadata,
                          "network_name",
                        );
                        const command = metadataValue(
                          execution.metadata,
                          "command",
                        );
                        const evidence = metadataValue(
                          execution.metadata,
                          "evidence_paths",
                        );
                        return (
                          <div
                            key={execution.id}
                            className="rounded-md border border-border bg-background px-3 py-3"
                          >
                            <div className="flex items-center justify-between gap-2">
                              <div className="text-sm font-semibold text-foreground">
                                {execution.profile}
                              </div>
                              <Badge variant={statusVariant(execution.status)}>
                                {execution.status}
                              </Badge>
                            </div>
                            <div className="mt-2 text-xs text-muted-foreground">
                              {execution.runtime} / {formatTime(execution.completed_at ?? execution.started_at)}
                            </div>
                            {image ? (
                              <div className="mt-2 text-xs text-muted-foreground">
                                image {image}
                              </div>
                            ) : null}
                            {networkMode ? (
                              <div className="mt-1 text-xs text-muted-foreground">
                                network{" "}
                                {networkMode === "custom" && networkName
                                  ? `${networkMode} (${networkName})`
                                  : networkMode}
                              </div>
                            ) : null}
                            {command ? (
                              <div className="mt-1 text-xs text-muted-foreground">
                                command {truncate(command)}
                              </div>
                            ) : null}
                            {evidence ? (
                              <div className="mt-1 text-xs text-muted-foreground">
                                evidence {truncate(evidence)}
                              </div>
                            ) : null}
                          </div>
                        );
                      })
                  ) : (
                    <p className="text-sm text-muted-foreground">
                      No execution traces recorded yet.
                    </p>
                  )}
                </div>
              </CardContent>
            </Card>

            <Card className="border-border bg-card">
              <CardContent className="space-y-4 p-4">
                <div className="flex items-center gap-2">
                  <FileBox className="h-4 w-4 text-muted-foreground" />
                  <h2 className="text-base font-semibold text-foreground">
                    Artifacts
                  </h2>
                </div>
                <div className="space-y-3">
                  {workspace?.artifacts.length ? (
                    workspace.artifacts.slice(0, 12).map((artifact) => {
                      const image = metadataValue(artifact.metadata, "image");
                      const networkMode = metadataValue(
                        artifact.metadata,
                        "network_mode",
                      );
                      const command = metadataValue(artifact.metadata, "command");
                      const evidence = metadataValue(
                        artifact.metadata,
                        "evidence_paths",
                      );
                      return (
                        <div
                          key={artifact.id}
                          className="rounded-md border border-border bg-background px-3 py-3"
                        >
                          <div className="flex items-center justify-between gap-2">
                            <div className="text-sm font-semibold text-foreground">
                              {artifact.name}
                            </div>
                            <Badge variant="outline">{artifact.kind}</Badge>
                          </div>
                          <div className="mt-2 text-xs text-muted-foreground">
                            created {formatTime(artifact.created_at)}
                          </div>
                          {artifact.content ? (
                            <div className="mt-2 text-xs leading-5 text-muted-foreground">
                              {truncate(artifact.content, 180)}
                            </div>
                          ) : null}
                          {image ? (
                            <div className="mt-2 text-xs text-muted-foreground">
                              image {image}
                            </div>
                          ) : null}
                          {networkMode ? (
                            <div className="mt-1 text-xs text-muted-foreground">
                              network {networkMode}
                            </div>
                          ) : null}
                          {command ? (
                            <div className="mt-1 text-xs text-muted-foreground">
                              command {truncate(command)}
                            </div>
                          ) : null}
                          {evidence ? (
                            <div className="mt-1 text-xs text-muted-foreground">
                              evidence {truncate(evidence)}
                            </div>
                          ) : null}
                        </div>
                      );
                    })
                  ) : (
                    <p className="text-sm text-muted-foreground">
                      No artifacts stored yet.
                    </p>
                  )}
                </div>
              </CardContent>
            </Card>
          </div>
        </div>

        {workspace ? (
          <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
            <Clock3 className="h-3.5 w-3.5" />
            created {formatTime(workspace.flow.created_at)}
            <ArrowRight className="h-3.5 w-3.5" />
            <CheckCircle2 className="h-3.5 w-3.5" />
            updated {formatTime(workspace.flow.updated_at)}
          </div>
        ) : null}
      </div>

      <ConfirmDialog
        open={confirmCancel}
        onCancel={() => setConfirmCancel(false)}
        onConfirm={handleCancelFlow}
        title="Cancel flow"
        message="Are you sure you want to cancel this flow? This action cannot be undone."
        confirmLabel="Cancel flow"
        variant="danger"
      />

      <ConfirmDialog
        open={confirmRejectId !== null}
        onCancel={() => setConfirmRejectId(null)}
        onConfirm={() => {
          if (confirmRejectId) {
            handleReview(confirmRejectId, false);
            setConfirmRejectId(null);
          }
        }}
        title="Reject approval"
        message="Are you sure you want to reject this approval?"
        confirmLabel="Reject"
        variant="danger"
      />
    </AppShell>
  );
}
