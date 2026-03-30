import AppShell from "@/components/layout/AppShell";

export default function NewScanLoading() {
  return (
    <AppShell>
      <div className="space-y-6 px-4 py-4 md:px-6 md:py-6 xl:px-8">
        <div className="h-8 w-48 animate-pulse rounded-md bg-muted" />
        <div className="max-w-xl space-y-4">
          <div className="h-11 animate-pulse rounded-lg bg-muted" />
          <div className="h-11 animate-pulse rounded-lg bg-muted" />
          <div className="h-24 animate-pulse rounded-lg bg-muted" />
          <div className="h-11 w-32 animate-pulse rounded-lg bg-muted" />
        </div>
      </div>
    </AppShell>
  );
}
