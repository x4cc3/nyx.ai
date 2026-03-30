"use client";

import type { ReactNode } from "react";
import { useState } from "react";
import ErrorBoundary from "@/components/common/ErrorBoundary";
import Drawer from "@/components/ui/Drawer";
import Sidebar from "./Sidebar";
import ContextPanel from "./ContextPanel";
import type { AgentGraph, ScanStatusData } from "@/lib/types";

interface AppShellProps {
  children: ReactNode;
  scan?: ScanStatusData | null;
  agentGraph?: AgentGraph | null;
  showContext?: boolean;

}

export default function AppShell({
  children,
  scan,
  agentGraph,
  showContext = false,
}: AppShellProps) {
  const [contextOpen, setContextOpen] = useState(false);

  return (
    <div className="min-h-dvh bg-transparent text-foreground md:flex">
      <Sidebar
        showContextButton={showContext}
        onOpenContext={() => setContextOpen(true)}
      />

      <div className="min-w-0 flex-1 md:flex">
        <main className="min-w-0 flex-1">
          <ErrorBoundary>{children}</ErrorBoundary>
        </main>

        {showContext ? (
          <>
            <aside className="hidden w-[320px] shrink-0 border-l border-border bg-card xl:block">
              <div className="sticky top-0 h-dvh overflow-y-auto">
                <ContextPanel scan={scan} agentGraph={agentGraph} />
              </div>
            </aside>

            <Drawer
              open={contextOpen}
              onOpenChange={setContextOpen}
              side="right"
              title="Context"
            >
              <ContextPanel scan={scan} agentGraph={agentGraph} />
            </Drawer>
          </>
        ) : null}
      </div>
    </div>
  );
}
