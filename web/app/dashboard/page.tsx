"use client";

export const dynamic = "force-dynamic";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import AppShell from "@/components/layout/AppShell";
import { Button, Card, CardContent, Skeleton } from "@/components/ui";
import { cn } from "@/lib/cn";
import { listApprovals, listFlows } from "@/lib/api";
import type { Approval, Flow } from "@/lib/types";
import {
  Activity,
  ArrowRight,
  CheckCircle2,
  Clock3,
  ShieldAlert,
} from "lucide-react";

function formatTime(value?: string) {
  if (!value) return "unknown";
  return new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value));
}

function statusTone(status: string) {
  switch (status) {
    case "running":
      return "text-primary";
    case "completed":
      return "text-emerald-400";
    case "failed":
      return "text-red-400";
    case "cancelled":
      return "text-amber-300";
    case "queued":
      return "text-amber-400";
    default:
      return "text-muted-foreground";
  }
}

export default function DashboardPage() {
  const [flows, setFlows] = useState<Flow[]>([]);
  const [approvals, setApprovals] = useState<Approval[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let active = true;

    async function load() {
      try {
        const [flowsPayload, approvalsPayload] = await Promise.all([
          listFlows(),
          listApprovals(),
        ]);
        if (!active) return;
        setFlows(flowsPayload.flows);
        setApprovals(approvalsPayload.approvals);
        setError(null);
      } catch (loadError) {
        if (!active) return;
        setError(
          loadError instanceof Error
            ? loadError.message
            : "dashboard load failed",
        );
      } finally {
        if (active) {
          setLoading(false);
        }
      }
    }

    void load();
    const timer = window.setInterval(() => void load(), 7000);
    return () => {
      active = false;
      window.clearInterval(timer);
    };
  }, []);

  const activeCount = useMemo(
    () =>
      flows.filter(
        (flow) => flow.status === "running" || flow.status === "queued",
      ).length,
    [flows],
  );
  const completedCount = useMemo(
    () =>
      flows.filter(
        (flow) => flow.status === "completed" || flow.status === "cancelled",
      ).length,
    [flows],
  );
  const pendingApprovals = useMemo(
    () => approvals.filter((approval) => approval.status === "pending"),
    [approvals],
  );

  return (
    <AppShell>
      <div className="space-y-6 px-4 py-4 md:px-6 md:py-6 xl:px-8">
        <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div className="space-y-2">
            <h1 className="text-2xl font-semibold text-foreground">
              Flow operations
            </h1>
            <p className="max-w-2xl text-sm leading-6 text-muted-foreground">
              View running work, review approvals, and jump back into a flow.
            </p>
          </div>
          <Button asChild>
            <Link href="/scans/new">New flow</Link>
          </Button>
        </div>

        <div className="grid gap-3 md:grid-cols-3">
          {[
            {
              label: "Active",
              value: activeCount,
              description: "Queued or running",
            },
            {
              label: "Approvals",
              value: pendingApprovals.length,
              description: "Pending review",
            },
            {
              label: "Closed",
              value: completedCount,
              description: "Completed flows",
            },
          ].map((item) => (
            <Card key={item.label} className="border-border bg-card">
              <CardContent className="space-y-2 p-4">
                <p className="text-sm font-medium text-foreground">{item.label}</p>
                <p className="text-3xl font-semibold text-foreground">{item.value}</p>
                <p className="text-sm text-muted-foreground">{item.description}</p>
              </CardContent>
            </Card>
          ))}
        </div>

        {error ? (
          <div className="rounded-md border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-200">
            {error}
          </div>
        ) : null}

        <Card className="border-border bg-card">
          <CardContent className="p-0">
            <div className="border-b border-border px-4 py-3">
              <h2 className="text-base font-semibold text-foreground">Flows</h2>
            </div>

            <div className="divide-y divide-border">
              {loading ? (
                Array.from({ length: 4 }).map((_, index) => (
                  <div key={index} className="px-4 py-4">
                    <Skeleton className="h-16 w-full rounded-md" />
                  </div>
                ))
              ) : flows.length === 0 ? (
                <div className="px-4 py-12 text-center">
                  <Activity className="mx-auto h-5 w-5 text-muted-foreground" />
                  <p className="mt-3 text-sm text-muted-foreground">
                    No flows yet.
                  </p>
                </div>
              ) : (
                flows.map((flow) => (
                  <Link
                    key={flow.id}
                    href={`/scans/${flow.id}`}
                    className="grid gap-3 px-4 py-4 transition-colors hover:bg-muted/30 xl:grid-cols-[minmax(0,1fr)_minmax(0,1.4fr)_160px_120px_32px]"
                  >
                    <div className="space-y-1">
                      <p className={cn("text-sm font-medium", statusTone(flow.status))}>
                        {flow.status}
                      </p>
                      <p className="text-sm font-semibold text-foreground">
                        {flow.name}
                      </p>
                      <p className="text-sm text-muted-foreground">{flow.target}</p>
                    </div>
                    <p className="text-sm leading-6 text-muted-foreground">
                      {flow.objective}
                    </p>
                    <div className="flex items-center gap-2 text-sm text-muted-foreground">
                      <Clock3 className="h-4 w-4" />
                      <span>{formatTime(flow.updated_at)}</span>
                    </div>
                    <div className="flex items-center gap-2 text-sm text-muted-foreground">
                      <CheckCircle2 className="h-4 w-4" />
                      <span>{flow.tenant_id}</span>
                    </div>
                    <div className="flex items-center justify-end text-muted-foreground">
                      <ArrowRight className="h-4 w-4" />
                    </div>
                  </Link>
                ))
              )}
            </div>
          </CardContent>
        </Card>

        <Card className="border-border bg-card">
          <CardContent className="space-y-4 p-4">
            <div className="flex items-center gap-2">
              <ShieldAlert className="h-4 w-4 text-muted-foreground" />
              <h2 className="text-base font-semibold text-foreground">
                Pending approvals
              </h2>
            </div>

            <div className="space-y-3">
              {pendingApprovals.length === 0 ? (
                <p className="text-sm text-muted-foreground">
                  No approvals waiting.
                </p>
              ) : (
                pendingApprovals.slice(0, 5).map((approval) => (
                  <div
                    key={approval.id}
                    className="rounded-md border border-border bg-background px-3 py-3"
                  >
                    <p className="text-sm font-medium text-foreground">
                      {approval.kind}
                    </p>
                    <p className="mt-1 text-sm leading-6 text-muted-foreground">
                      {approval.reason}
                    </p>
                  </div>
                ))
              )}
            </div>
          </CardContent>
        </Card>
      </div>
    </AppShell>
  );
}
