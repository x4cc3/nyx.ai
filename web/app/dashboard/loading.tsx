import AppShell from "@/components/layout/AppShell";

export default function DashboardLoading() {
  return (
    <AppShell>
      <div className="space-y-6 px-4 py-4 md:px-6 md:py-6 xl:px-8">
        <div className="h-8 w-40 animate-pulse rounded-md bg-muted" />
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <div
              key={i}
              className="h-32 animate-pulse rounded-lg border border-border bg-card"
            />
          ))}
        </div>
        <div className="h-64 animate-pulse rounded-lg border border-border bg-card" />
      </div>
    </AppShell>
  );
}
