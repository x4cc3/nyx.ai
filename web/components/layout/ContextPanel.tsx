"use client";

import { useMemo } from "react";
import { Badge } from "@/components/ui";
import { cn } from "@/lib/cn";
import { formatStatus } from "@/lib/format";
import type { AgentGraph, ScanStatusData } from "@/lib/types";
import { Activity, GitBranch } from "lucide-react";
import { useLanguage } from "@/components/i18n/LanguageProvider";

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
      return "secondary" as const;
  }
}

export default function ContextPanel({
  scan,
  agentGraph,
}: {
  scan?: ScanStatusData | null;
  agentGraph?: AgentGraph | null;
}) {
  const { t } = useLanguage();
  const agentCounts = useMemo(() => {
    const base = { running: 0, completed: 0, failed: 0, waiting: 0 };
    if (!agentGraph?.nodes?.length) return base;
    for (const n of agentGraph.nodes) {
      const s = String(n.status || "");
      if (s === "running") base.running += 1;
      else if (s === "completed") base.completed += 1;
      else if (s === "failed") base.failed += 1;
      else base.waiting += 1;
    }
    return base;
  }, [agentGraph]);
  const progressPct = useMemo(() => {
    const cur = scan?.current_step;
    const total = scan?.total_steps;
    if (!cur || !total) return 0;
    return Math.max(0, Math.min(100, Math.round((cur / total) * 100)));
  }, [scan?.current_step, scan?.total_steps]);

  return (
    <div className="space-y-5 p-4">
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold text-foreground">Context</h2>
        {scan?.scan_id ? (
          <span className="font-mono text-xs text-muted-foreground">
            {scan.scan_id.slice(0, 8)}
          </span>
        ) : null}
      </div>

      <section className="space-y-2">
        <div className="flex items-center gap-2 text-sm font-medium text-foreground">
          <Activity className="h-4 w-4 text-muted-foreground" />
          <span>{t("right_rail.status")}</span>
        </div>
        {scan ? (
          <div className="space-y-3 rounded-md border border-border bg-background px-3 py-3">
            <div className="flex items-center justify-between gap-3">
              <div className="text-sm text-muted-foreground">{t("right_rail.scan")}</div>
              <Badge variant={statusVariant(scan.status)}>
                {formatStatus(scan.status, t)}
              </Badge>
            </div>
            <div className="flex items-center justify-between gap-3 text-sm">
              <span className="text-muted-foreground">{t("right_rail.current_step")}</span>
              <span className="font-mono text-foreground">
                {scan.current_step != null ? `#${scan.current_step}` : "—"}
              </span>
            </div>
            {scan.current_step != null && scan.total_steps != null ? (
              <div className="overflow-hidden rounded-sm border border-border bg-background">
                <div className="h-1.5 bg-primary/70" style={{ width: `${progressPct}%` }} />
              </div>
            ) : null}
            {scan.pending_reason ? (
              <div className="text-sm text-muted-foreground">{scan.pending_reason}</div>
            ) : null}
          </div>
        ) : (
          <div className="rounded-md border border-border bg-background px-3 py-3 text-sm text-muted-foreground">
            {t("right_rail.not_selected")}
          </div>
        )}
      </section>

      <section className="space-y-2">
        <div className="flex items-center gap-2 text-sm font-medium text-foreground">
          <GitBranch className="h-4 w-4 text-muted-foreground" />
          <span>{t("right_rail.tabs.graph")}</span>
        </div>
        {agentGraph && agentGraph.nodes.length > 0 ? (
          <div className="space-y-3 rounded-md border border-border bg-background px-3 py-3">
            <div className="grid grid-cols-2 gap-2 text-sm md:grid-cols-4">
              <div className="rounded-md border border-border bg-card px-3 py-2">
                <div className="text-xs text-muted-foreground">Running</div>
                <div className="mt-1 font-medium text-foreground">{agentCounts.running}</div>
              </div>
              <div className="rounded-md border border-border bg-card px-3 py-2">
                <div className="text-xs text-muted-foreground">Completed</div>
                <div className="mt-1 font-medium text-foreground">
                  {agentCounts.completed}
                </div>
              </div>
              <div className="rounded-md border border-border bg-card px-3 py-2">
                <div className="text-xs text-muted-foreground">Failed</div>
                <div className="mt-1 font-medium text-foreground">{agentCounts.failed}</div>
              </div>
              <div className="rounded-md border border-border bg-card px-3 py-2">
                <div className="text-xs text-muted-foreground">Waiting</div>
                <div className="mt-1 font-medium text-foreground">{agentCounts.waiting}</div>
              </div>
            </div>
            <div className="space-y-2">
              {agentGraph.nodes.slice(0, 12).map((node) => (
                <div
                  key={node.agent_id}
                  className="flex items-center gap-2 rounded-md border border-border bg-card px-3 py-2"
                >
                  <span
                    className={cn(
                      "h-2 w-2 rounded-full",
                      node.status === "running"
                        ? "bg-primary"
                        : node.status === "completed"
                          ? "bg-nyx-success"
                          : node.status === "failed"
                            ? "bg-nyx-hot"
                            : "bg-muted-foreground/40"
                    )}
                  />
                  <span className="truncate text-sm text-foreground">{node.name}</span>
                  <span className="ml-auto text-xs text-muted-foreground">{node.role}</span>
                </div>
              ))}
            </div>
          </div>
        ) : (
          <div className="rounded-md border border-border bg-background px-3 py-3 text-sm text-muted-foreground">
            {t("agent_graph.empty")}
          </div>
        )}
      </section>
    </div>
  );
}
