import AppShell from "@/components/layout/AppShell";

export default function ScanLoading() {
  return (
    <AppShell showContext>
      <div className="space-y-4 px-4 py-4 md:px-6 md:py-6 xl:px-8">
        <div className="h-8 w-56 animate-pulse rounded-md bg-muted" />
        <div className="h-24 animate-pulse rounded-lg border border-border bg-card" />
        <div className="h-24 animate-pulse rounded-lg border border-border bg-card" />
        <div className="h-24 animate-pulse rounded-lg border border-border bg-card" />
      </div>
    </AppShell>
  );
}
