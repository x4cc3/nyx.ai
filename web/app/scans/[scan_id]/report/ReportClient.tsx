"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import AppShell from "@/components/layout/AppShell";
import { Badge, Button, Card, CardContent, Skeleton } from "@/components/ui";
import { downloadReport, getWorkspace } from "@/lib/api";
import type { AgentGraph, ScanStatusData, Workspace } from "@/lib/types";
import {
  ArrowLeft,
  Download,
  FileText,
  ShieldAlert,
  Target,
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

function statusVariant(status?: string) {
  switch (status) {
    case "running":
      return "accent" as const;
    case "completed":
      return "success" as const;
    case "failed":
      return "hot" as const;
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

export default function ReportClient({ scanId }: { scanId: string }) {
  const [workspace, setWorkspace] = useState<Workspace | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [downloading, setDownloading] = useState<"markdown" | "json" | "pdf" | null>(
    null,
  );

  useEffect(() => {
    let active = true;

    async function load() {
      try {
        const nextWorkspace = await getWorkspace(scanId);
        if (!active) return;
        setWorkspace(nextWorkspace);
        setError(null);
      } catch (loadError) {
        if (!active) return;
        setError(loadError instanceof Error ? loadError.message : "report load failed");
      } finally {
        if (active) {
          setLoading(false);
        }
      }
    }

    void load();
    return () => {
      active = false;
    };
  }, [scanId]);

  const scan = useMemo(() => toScanStatus(workspace), [workspace]);
  const agentGraph = useMemo(() => toAgentGraph(workspace), [workspace]);

  async function handleDownload(format: "markdown" | "json" | "pdf") {
    setDownloading(format);
    try {
      await downloadReport(scanId, format);
    } catch (downloadError) {
      setError(
        downloadError instanceof Error
          ? downloadError.message
          : "report download failed",
      );
    } finally {
      setDownloading(null);
    }
  }

  return (
    <AppShell scan={scan} agentGraph={agentGraph} showContext>
      <div className="space-y-6 px-4 py-4 md:px-6 md:py-6 xl:px-8">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div className="space-y-2">
            <div className="text-sm text-muted-foreground">Report</div>
            <h1 className="text-2xl font-semibold text-foreground">
              {workspace?.flow.name ?? "Flow report"}
            </h1>
            <div className="flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
              <Badge variant={statusVariant(workspace?.flow.status)}>
                {workspace?.flow.status ?? "loading"}
              </Badge>
              <span className="font-mono">{workspace?.flow.target ?? scanId}</span>
            </div>
          </div>

          <div className="flex flex-wrap gap-2">
            <Button asChild variant="secondary" size="sm">
              <Link href={`/scans/${scanId}`}>
                <ArrowLeft className="mr-1.5 h-4 w-4" />
                Back
              </Link>
            </Button>
            <Button
              size="sm"
              variant="secondary"
              onClick={() => handleDownload("markdown")}
              disabled={downloading !== null}
            >
              <Download className="mr-1.5 h-4 w-4" />
              MD
            </Button>
            <Button
              size="sm"
              variant="secondary"
              onClick={() => handleDownload("json")}
              disabled={downloading !== null}
            >
              <Download className="mr-1.5 h-4 w-4" />
              JSON
            </Button>
            <Button
              size="sm"
              onClick={() => handleDownload("pdf")}
              disabled={downloading !== null}
            >
              <Download className="mr-1.5 h-4 w-4" />
              PDF
            </Button>
          </div>
        </div>

        {error ? (
          <div className="rounded-md border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-200">
            {error}
          </div>
        ) : null}

        <Card className="border-border bg-card">
          <CardContent className="space-y-5 p-4 md:p-5">
            {loading && !workspace ? (
              <div className="space-y-3">
                <Skeleton className="h-8 w-1/2 rounded-md" />
                <Skeleton className="h-28 w-full rounded-md" />
              </div>
            ) : (
              <>
                <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
                  {[
                    { label: "Findings", value: workspace?.findings.length ?? 0 },
                    { label: "Actions", value: workspace?.actions.length ?? 0 },
                    { label: "Artifacts", value: workspace?.artifacts.length ?? 0 },
                    { label: "Agents", value: workspace?.agents.length ?? 0 },
                  ].map((item) => (
                    <div
                      key={item.label}
                      className="rounded-md border border-border bg-background px-4 py-4"
                    >
                      <p className="text-sm font-medium text-foreground">
                        {item.label}
                      </p>
                      <p className="mt-2 text-2xl font-semibold text-foreground">
                        {item.value}
                      </p>
                    </div>
                  ))}
                </div>

                <div className="grid gap-4 xl:grid-cols-[1.15fr,0.85fr]">
                  <div className="space-y-4">
                    <div className="rounded-md border border-border bg-background px-4 py-4">
                      <div className="mb-2 flex items-center gap-2">
                        <Target className="h-4 w-4 text-muted-foreground" />
                        <h2 className="text-base font-semibold text-foreground">
                          Objective
                        </h2>
                      </div>
                      <div className="text-sm leading-6 text-muted-foreground">
                        {workspace?.flow.objective || "No objective recorded."}
                      </div>
                    </div>

                    <div className="rounded-md border border-border bg-background px-4 py-4">
                      <div className="mb-3 flex items-center gap-2">
                        <ShieldAlert className="h-4 w-4 text-muted-foreground" />
                        <h2 className="text-base font-semibold text-foreground">
                          Findings
                        </h2>
                      </div>
                      <div className="space-y-3">
                        {workspace?.findings.length ? (
                          workspace.findings.map((finding) => (
                            <div
                              key={finding.id}
                              className="rounded-md border border-border bg-card px-4 py-4"
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
                              <div className="mt-3 text-xs text-muted-foreground">
                                captured {formatTime(finding.created_at)}
                              </div>
                            </div>
                          ))
                        ) : (
                          <p className="text-sm text-muted-foreground">
                            No findings recorded in this flow yet.
                          </p>
                        )}
                      </div>
                    </div>
                  </div>

                  <div className="space-y-4">
                    <div className="rounded-md border border-border bg-background px-4 py-4">
                      <div className="mb-3 flex items-center gap-2">
                        <FileText className="h-4 w-4 text-muted-foreground" />
                        <h2 className="text-base font-semibold text-foreground">
                          Evidence
                        </h2>
                      </div>
                      <div className="space-y-3">
                        {workspace?.artifacts.length ? (
                          workspace.artifacts.map((artifact) => (
                            <div
                              key={artifact.id}
                              className="rounded-md border border-border bg-card px-3 py-3"
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
                            </div>
                          ))
                        ) : (
                          <p className="text-sm text-muted-foreground">
                            No artifacts attached to this flow.
                          </p>
                        )}
                      </div>
                    </div>

                    <div className="rounded-md border border-border bg-background px-4 py-4">
                      <h2 className="mb-3 text-base font-semibold text-foreground">
                        Recent actions
                      </h2>
                      <div className="space-y-3">
                        {workspace?.actions.length ? (
                          workspace.actions
                            .slice()
                            .reverse()
                            .slice(0, 8)
                            .map((action) => (
                              <div
                                key={action.id}
                                className="rounded-md border border-border bg-card px-3 py-3"
                              >
                                <div className="flex items-center justify-between gap-2">
                                  <div className="text-sm font-semibold text-foreground">
                                    {action.function_name}
                                  </div>
                                  <Badge variant={statusVariant(action.status)}>
                                    {action.status}
                                  </Badge>
                                </div>
                                <div className="mt-2 text-xs text-muted-foreground">
                                  {action.agent_role} / {formatTime(action.updated_at)}
                                </div>
                              </div>
                            ))
                        ) : (
                          <p className="text-sm text-muted-foreground">
                            No completed actions yet.
                          </p>
                        )}
                      </div>
                    </div>
                  </div>
                </div>
              </>
            )}
          </CardContent>
        </Card>
      </div>
    </AppShell>
  );
}
