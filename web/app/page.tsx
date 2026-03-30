export const dynamic = "force-dynamic";

import Link from "next/link";
import Trans from "@/components/i18n/Trans";
import { Button, Card, CardContent } from "@/components/ui";
import { NyxLogo } from "@/components/ui/NyxLogo";
import {
  BookOpen,
  CheckCircle,
  FileCheck,
  FileText,
  Github,
  LayoutDashboard,
  LogIn,
  Plus,
  Radar,
  Search,
  ShieldCheck,
} from "lucide-react";

const kpis = [
  {
    icon: NyxLogo,
    labelKey: "landing.kpi.scope.label",
    valueKey: "landing.kpi.scope.value",
    captionKey: "landing.kpi.scope.caption",
  },
  {
    icon: FileCheck,
    labelKey: "landing.kpi.evidence.label",
    valueKey: "landing.kpi.evidence.value",
    captionKey: "landing.kpi.evidence.caption",
  },
  {
    icon: ShieldCheck,
    labelKey: "landing.kpi.guardrails.label",
    valueKey: "landing.kpi.guardrails.value",
    captionKey: "landing.kpi.guardrails.caption",
  },
] as const;

const features = [
  {
    titleKey: "landing.feature.guided.title",
    bodyKey: "landing.feature.guided.body",
    items: [
      "landing.feature.guided.item1",
      "landing.feature.guided.item2",
      "landing.feature.guided.item3",
    ],
  },
  {
    titleKey: "landing.feature.oversight.title",
    bodyKey: "landing.feature.oversight.body",
    items: [
      "landing.feature.oversight.item1",
      "landing.feature.oversight.item2",
      "landing.feature.oversight.item3",
    ],
  },
] as const;

const timeline = [
  {
    index: "01",
    icon: Search,
    titleKey: "landing.timeline.recon.title",
    bodyKey: "landing.timeline.recon.body",
  },
  {
    index: "02",
    icon: Radar,
    titleKey: "landing.timeline.scan.title",
    bodyKey: "landing.timeline.scan.body",
  },
  {
    index: "03",
    icon: CheckCircle,
    titleKey: "landing.timeline.validate.title",
    bodyKey: "landing.timeline.validate.body",
  },
  {
    index: "04",
    icon: FileText,
    titleKey: "landing.timeline.report.title",
    bodyKey: "landing.timeline.report.body",
  },
] as const;

const demoLines = [
  "DNS enumeration -> 14 subdomains discovered",
  "Port scan -> 443/tcp open (nginx 1.24)",
  "Nuclei CVE-2023-44487 -> candidate",
  "Dalfox XSS scan -> 3 reflected params",
  "CVE-2023-44487 verified",
  "XSS payload confirmed on /search?q=",
  "Report synthesised -> 2 critical, 1 high",
] as const;

const topNavItems = [
  {
    href: "/dashboard",
    icon: LayoutDashboard,
    label: "Dashboard",
    tone: "default" as const,
  },
  {
    href: "/scans/new",
    icon: Plus,
    label: "New scan",
    tone: "default" as const,
  },
  {
    href: "/login",
    icon: LogIn,
    label: "Sign in",
    tone: "primary" as const,
  },
] as const;

function DemoLog() {
  return (
    <Card className="border-border bg-card">
      <CardContent className="p-0">
        <div className="border-b border-border px-4 py-3">
          <p className="text-sm font-medium text-foreground">Recent run</p>
        </div>
        <div className="space-y-0 border-l-0 px-4 py-2">
          {demoLines.map((line) => (
            <div
              key={line}
              className="border-b border-border/60 py-2 font-mono text-xs leading-5 text-muted-foreground last:border-b-0"
            >
              {line}
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}

function GuardrailList() {
  return (
    <Card className="border-border bg-card">
      <CardContent className="space-y-4 p-4">
        <h2 className="text-base font-semibold text-foreground">
          <Trans id="left_rail.guardrails" />
        </h2>
        <ul className="space-y-2 text-sm text-muted-foreground">
          <li>
            <Trans id="left_rail.guardrail.item1" />
          </li>
          <li>
            <Trans id="left_rail.guardrail.item2" />
          </li>
          <li>
            <Trans id="left_rail.guardrail.item3" />
          </li>
        </ul>
      </CardContent>
    </Card>
  );
}

export default function WelcomePage() {
  return (
    <div className="relative min-h-dvh overflow-hidden bg-transparent text-foreground">
      <header className="relative z-10 border-b border-border/80 bg-background/72 backdrop-blur-sm">
        <div className="mx-auto flex w-full max-w-6xl flex-wrap items-center justify-between gap-3 px-4 py-4 md:px-6">
          <Link href="/" className="flex items-center gap-3">
            <NyxLogo className="h-6 w-6 text-primary" />
            <span className="text-sm font-semibold text-foreground">NYX</span>
          </Link>

          <nav className="flex w-full text-sm sm:w-auto">
            <div className="flex w-full items-center gap-1 rounded-lg border border-border/80 bg-card/88 p-1 sm:w-auto">
              {topNavItems.map(({ href, icon: Icon, label, tone }) => (
                <Link
                  key={href}
                  href={href}
                  className={
                    tone === "primary"
                      ? "inline-flex flex-1 items-center justify-center gap-2 rounded-md bg-primary px-3 py-2 font-medium text-primary-foreground transition-colors hover:bg-primary/90 sm:flex-none"
                      : "inline-flex flex-1 items-center justify-center gap-2 rounded-md px-3 py-2 text-muted-foreground transition-colors hover:bg-background/70 hover:text-foreground sm:flex-none"
                  }
                >
                  <Icon className="h-4 w-4 shrink-0" />
                  <span>{label}</span>
                </Link>
              ))}
            </div>
          </nav>
        </div>
      </header>

      <main className="relative z-10 mx-auto flex w-full max-w-6xl flex-col gap-12 px-4 py-8 md:px-6 md:py-10">
        <section className="grid gap-8 lg:grid-cols-[minmax(0,1.3fr)_minmax(0,0.9fr)]">
          <div className="space-y-6">
            <div className="space-y-3">
              <h1 className="text-3xl font-semibold tracking-tight text-foreground sm:text-4xl">
                <Trans id="landing.hero.title" />
              </h1>
              <p className="max-w-2xl text-base leading-7 text-muted-foreground">
                <Trans id="landing.hero.subtitle" />
              </p>
            </div>

            <div className="flex flex-wrap gap-3">
              <Button asChild>
                <Link href="/scans/new">
                  <Trans id="button.start_new_scan" />
                </Link>
              </Button>
              <Button asChild variant="outline">
                <Link href="/dashboard">
                  <Trans id="button.open_dashboard" />
                </Link>
              </Button>
              <Button asChild variant="secondary">
                <Link href="/login">
                  <Trans id="button.login" />
                </Link>
              </Button>
            </div>

            <div className="grid gap-3 sm:grid-cols-3">
              {kpis.map(({ icon: Icon, labelKey, valueKey, captionKey }) => (
                <Card
                  key={labelKey}
                  className="border-border bg-card/88 backdrop-blur-[2px]"
                >
                  <CardContent className="space-y-3 p-4">
                    <div className="flex items-center gap-2">
                      <Icon className="h-4 w-4 text-primary" />
                      <p className="text-sm font-medium text-foreground">
                        <Trans id={labelKey} />
                      </p>
                    </div>
                    <p className="text-sm text-foreground">
                      <Trans id={valueKey} />
                    </p>
                    <p className="text-sm leading-6 text-muted-foreground">
                      <Trans id={captionKey} />
                    </p>
                  </CardContent>
                </Card>
              ))}
            </div>
          </div>

          <div className="space-y-3">
            <DemoLog />
            <GuardrailList />
          </div>
        </section>

        <section className="grid gap-6 lg:grid-cols-[220px_minmax(0,1fr)]">
          <div>
            <h2 className="text-xl font-semibold text-foreground">
              <Trans id="landing.section.lifecycle.title" />
            </h2>
          </div>
          <div className="grid gap-3 md:grid-cols-2">
            {features.map(({ titleKey, bodyKey, items }) => (
              <Card key={titleKey} className="border-border bg-card">
                <CardContent className="space-y-4 p-4">
                  <div className="space-y-2">
                    <h3 className="text-sm font-semibold text-foreground">
                      <Trans id={titleKey} />
                    </h3>
                    <p className="text-sm leading-6 text-muted-foreground">
                      <Trans id={bodyKey} />
                    </p>
                  </div>
                  <ul className="space-y-2 text-sm text-muted-foreground">
                    {items.map((itemKey) => (
                      <li key={itemKey}>
                        <Trans id={itemKey} />
                      </li>
                    ))}
                  </ul>
                </CardContent>
              </Card>
            ))}
          </div>
        </section>

        <section className="space-y-4">
          <h2 className="text-xl font-semibold text-foreground">
            <Trans id="landing.section.workflow.title" />
          </h2>
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
            {timeline.map((item) => (
              <Card key={item.index} className="border-border bg-card">
                <CardContent className="space-y-3 p-4">
                  <div className="flex items-center gap-2 text-sm font-medium text-foreground">
                    <span className="font-mono text-muted-foreground">
                      {item.index}
                    </span>
                    <item.icon className="h-4 w-4 text-primary" />
                  </div>
                  <h3 className="text-sm font-semibold text-foreground">
                    <Trans id={item.titleKey} />
                  </h3>
                  <p className="text-sm leading-6 text-muted-foreground">
                    <Trans id={item.bodyKey} />
                  </p>
                </CardContent>
              </Card>
            ))}
          </div>
        </section>

        <section className="rounded-lg border border-border bg-card">
          <div className="flex flex-col gap-4 p-4 md:flex-row md:items-center md:justify-between">
            <div className="space-y-2">
              <h2 className="text-lg font-semibold text-foreground">
                <Trans id="landing.callout.title" />
              </h2>
              <p className="max-w-2xl text-sm leading-6 text-muted-foreground">
                <Trans id="landing.callout.body" />
              </p>
            </div>
            <div className="flex flex-wrap gap-3">
              <Button asChild>
                <Link href="/scans/new">
                  <Trans id="button.launch_scan" />
                </Link>
              </Button>
              <Button asChild variant="outline">
                <Link href="/dashboard">
                  <Trans id="button.view_dashboard" />
                </Link>
              </Button>
            </div>
          </div>
        </section>
      </main>

      <footer className="relative z-10 border-t border-border/80 bg-background/72 backdrop-blur-sm">
        <div className="mx-auto flex w-full max-w-6xl flex-col gap-4 px-4 py-6 md:px-6 md:flex-row md:items-center md:justify-between">
          <div className="flex items-center gap-3">
            <NyxLogo className="h-5 w-5 text-primary" />
            <span className="text-sm font-medium text-foreground">NYX</span>
          </div>

          <nav className="flex flex-wrap items-center gap-x-4 gap-y-2 text-sm text-muted-foreground">
            <Link href="/scans/new" className="transition-colors hover:text-foreground">
              New scan
            </Link>
            <Link href="/dashboard" className="transition-colors hover:text-foreground">
              Dashboard
            </Link>
            <Link href="/login" className="transition-colors hover:text-foreground">
              Sign in
            </Link>
            <Link href="/register" className="transition-colors hover:text-foreground">
              Register
            </Link>
          </nav>

          <div className="flex items-center gap-4 text-sm text-muted-foreground">
            <a
              href={process.env.NEXT_PUBLIC_GITHUB_URL || "https://github.com/nyx-ai/nyx"}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 transition-colors hover:text-foreground"
            >
              <Github className="h-4 w-4" />
              <span>GitHub</span>
            </a>
            <a
              href="/docs"
              className="inline-flex items-center gap-2 transition-colors hover:text-foreground"
            >
              <BookOpen className="h-4 w-4" />
              <span>Docs</span>
            </a>
          </div>
        </div>
      </footer>
    </div>
  );
}
