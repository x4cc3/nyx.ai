"use client";

export const dynamic = "force-dynamic";

import { useState } from "react";
import { useRouter } from "next/navigation";
import AppShell from "@/components/layout/AppShell";
import { Button, Card, CardContent, Input, Textarea } from "@/components/ui";
import { ApiError, createFlow, startFlow } from "@/lib/api";
import { Loader2, ShieldCheck } from "lucide-react";

function normalizeTargetUrl(raw: string): string {
  let value = raw.trim();
  if (!value) return "";
  if (value.startsWith("//")) value = value.slice(2);
  if (!value.includes("://")) value = `https://${value}`;
  return value;
}

function isValidTargetUrl(value: string): boolean {
  if (!value) return false;
  try {
    const parsed = new URL(value);
    return parsed.protocol === "http:" || parsed.protocol === "https:";
  } catch {
    return false;
  }
}

export default function NewScanPage() {
  const router = useRouter();
  const [name, setName] = useState("");
  const [target, setTarget] = useState("");
  const [objective, setObjective] = useState("");
  const [autoStart, setAutoStart] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    setError(null);
    setFieldErrors({});

    const normalizedTarget = normalizeTargetUrl(target);
    const nextFieldErrors: Record<string, string> = {};
    if (!name.trim()) {
      nextFieldErrors.name = "Flow name is required.";
    }
    if (!isValidTargetUrl(normalizedTarget)) {
      nextFieldErrors.target = "Enter a valid http(s) target URL.";
    }
    if (!objective.trim()) {
      nextFieldErrors.objective = "Objective is required.";
    }
    if (Object.keys(nextFieldErrors).length > 0) {
      setFieldErrors(nextFieldErrors);
      setSubmitting(false);
      return;
    }

    try {
      const flow = await createFlow({
        name: name.trim(),
        target: normalizedTarget,
        objective: objective.trim(),
      });

      if (autoStart) {
        await startFlow(flow.id);
      }

      router.push(`/scans/${flow.id}`);
    } catch (submitError) {
      if (submitError instanceof ApiError) {
        if (submitError.fieldErrors) {
          setFieldErrors(submitError.fieldErrors);
        }
        setError(submitError.message);
      } else {
        setError("Unable to create flow right now.");
      }
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <AppShell>
      <div className="space-y-6 px-4 py-4 md:px-6 md:py-6 xl:px-8">
        <div className="space-y-2">
          <h1 className="text-2xl font-semibold text-foreground">
            Create flow
          </h1>
          <p className="max-w-2xl text-sm leading-6 text-muted-foreground">
            Set the target, define the objective, and decide whether the run should start immediately.
          </p>
        </div>

        <div className="grid gap-4 xl:grid-cols-[minmax(0,1.2fr)_320px]">
          <Card className="border-border bg-card">
            <CardContent className="p-5">
              <form onSubmit={onSubmit} className="space-y-5">
                <div className="grid gap-4 md:grid-cols-2">
                  <div className="space-y-2">
                    <label className="text-sm font-medium text-foreground">
                      Flow name
                    </label>
                    <Input
                      value={name}
                      onChange={(e) => setName(e.target.value)}
                      placeholder="Authenticated perimeter review"
                      className="font-mono"
                      required
                    />
                    {fieldErrors.name ? (
                      <p className="text-sm text-red-300">{fieldErrors.name}</p>
                    ) : null}
                  </div>
                  <div className="space-y-2">
                    <label className="text-sm font-medium text-foreground">
                      Target
                    </label>
                    <Input
                      value={target}
                      onChange={(e) => setTarget(e.target.value)}
                      placeholder="https://app.example.com"
                      className="font-mono"
                      required
                    />
                    {fieldErrors.target ? (
                      <p className="text-sm text-red-300">{fieldErrors.target}</p>
                    ) : null}
                  </div>
                </div>

                <div className="space-y-2">
                  <label className="text-sm font-medium text-foreground">
                    Objective
                  </label>
                  <Textarea
                    value={objective}
                    onChange={(e) => setObjective(e.target.value)}
                    placeholder="Map auth boundaries, validate exposed attack paths, and retain evidence."
                    rows={8}
                    className="font-mono"
                    required
                  />
                  {fieldErrors.objective ? (
                    <p className="text-sm text-red-300">{fieldErrors.objective}</p>
                  ) : null}
                </div>

                <label className="flex items-center gap-3 rounded-md border border-border bg-background px-4 py-3 text-sm text-muted-foreground">
                  <input
                    checked={autoStart}
                    onChange={(e) => setAutoStart(e.target.checked)}
                    type="checkbox"
                  />
                  Start immediately after creation
                </label>

                {error ? (
                  <div className="rounded-md border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-200">
                    {error}
                  </div>
                ) : null}

                <Button type="submit" disabled={submitting}>
                  {submitting ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : null}
                  {submitting ? "Submitting..." : "Create flow"}
                </Button>
              </form>
            </CardContent>
          </Card>

          <Card className="border-border bg-card">
            <CardContent className="space-y-4 p-5">
              <h2 className="text-base font-semibold text-foreground">
                Before you start
              </h2>
              <p className="text-sm leading-6 text-muted-foreground">
                Each flow stores tasks, artifacts, approvals, and execution traces in one place.
              </p>
              <p className="text-sm leading-6 text-muted-foreground">
                Terminal, browser, file, and search actions execute through the Go function gateway and executor manager.
              </p>
              <div className="rounded-md border border-border bg-background px-4 py-3 text-sm text-muted-foreground">
                <div className="flex items-center gap-2 text-foreground">
                  <ShieldCheck className="h-4 w-4" />
                  <span className="font-medium">Approval mode is still enforced.</span>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </AppShell>
  );
}
