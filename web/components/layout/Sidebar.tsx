"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useState } from "react";
import Drawer from "@/components/ui/Drawer";
import { NyxLogo } from "@/components/ui/NyxLogo";
import { useCurrentUser } from "@/hooks/useCurrentUser";
import { useLanguage } from "@/components/i18n/LanguageProvider";
import { cn } from "@/lib/cn";
import {
  LogIn,
  LogOut,
  Menu,
  PanelRightOpen,
  Plus,
  Radar,
  Settings2,
} from "lucide-react";

const navItems = [
  { href: "/dashboard", icon: Radar, labelKey: "button.dashboard" },
  { href: "/scans/new", icon: Plus, labelKey: "button.new_scan" },
  { href: "/settings", icon: Settings2, labelKey: "button.settings" },
] as const;

function SidebarNav({
  pathname,
  onNavigate,
}: {
  pathname: string | null;
  onNavigate?: () => void;
}) {
  const { t } = useLanguage();

  return (
    <nav className="space-y-1">
      {navItems.map(({ href, icon: Icon, labelKey }) => {
        const fallbackLabel =
          href === "/dashboard"
            ? "Dashboard"
            : href === "/scans/new"
              ? "New scan"
              : "Settings";
        const translated = t(labelKey);
        const label = translated === labelKey ? fallbackLabel : translated;
        const isActive = pathname !== null && (pathname === href || pathname.startsWith(href + "/"));

        return (
          <Link
            key={href}
            href={href}
            onClick={onNavigate}
            className={cn(
              "flex items-center gap-3 rounded-md px-3 py-2 text-sm transition-colors",
              isActive
                ? "bg-muted text-foreground"
                : "text-muted-foreground hover:bg-muted/60 hover:text-foreground",
            )}
            aria-current={isActive ? "page" : undefined}
          >
            <Icon className="h-4 w-4 shrink-0" />
            <span>{label}</span>
          </Link>
        );
      })}
    </nav>
  );
}

export default function Sidebar({
  showContextButton = false,
  onOpenContext,
}: {
  showContextButton?: boolean;
  onOpenContext?: () => void;
}) {
  const pathname = usePathname();
  const router = useRouter();
  const { t } = useLanguage();
  const { email: meEmail, logout } = useCurrentUser();
  const [mobileOpen, setMobileOpen] = useState(false);

  const [logoutError, setLogoutError] = useState<string | null>(null);

  async function handleLogout() {
    setLogoutError(null);
    const loggedOut = await logout();
    if (!loggedOut) {
      setLogoutError("Logout failed. Please try again.");
      return;
    }
    setMobileOpen(false);
    router.push("/login");
  }

  const sidebarBody = (
    <div className="flex h-full flex-col">
      <div className="border-b border-border px-4 py-4">
        <Link href="/dashboard" className="flex items-center gap-3">
          <NyxLogo className="h-6 w-6 text-primary" />
          <span className="text-sm font-semibold text-foreground">NYX</span>
        </Link>
      </div>

      <div className="flex-1 space-y-6 px-3 py-4">
        <SidebarNav pathname={pathname} onNavigate={() => setMobileOpen(false)} />

        {showContextButton && onOpenContext ? (
          <button
            type="button"
            onClick={() => {
              setMobileOpen(false);
              onOpenContext();
            }}
            className="flex w-full items-center gap-3 rounded-md px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-muted/60 hover:text-foreground xl:hidden"
          >
            <PanelRightOpen className="h-4 w-4 shrink-0" />
            <span>{t("button.context") || "Context"}</span>
          </button>
        ) : null}
      </div>

      <div className="border-t border-border px-3 py-4">
        {meEmail ? (
          <div className="space-y-3">
            <div className="rounded-md border border-border bg-background px-3 py-2">
              <p className="truncate text-sm text-foreground">{meEmail}</p>
            </div>
            {logoutError && (
              <p className="px-3 text-xs text-destructive">{logoutError}</p>
            )}
            <button
              type="button"
              onClick={handleLogout}
              className="flex w-full items-center gap-3 rounded-md px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-muted/60 hover:text-foreground"
            >
              <LogOut className="h-4 w-4 shrink-0" />
              <span>{t("button.logout")}</span>
            </button>
          </div>
        ) : (
          <Link
            href="/login"
            onClick={() => setMobileOpen(false)}
            className="flex items-center gap-3 rounded-md px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-muted/60 hover:text-foreground"
          >
            <LogIn className="h-4 w-4 shrink-0" />
            <span>{t("button.login")}</span>
          </Link>
        )}
      </div>
    </div>
  );

  return (
    <>
      <header className="sticky top-0 z-20 flex h-14 items-center gap-3 border-b border-border bg-background px-4 md:hidden">
        <button
          type="button"
          onClick={() => setMobileOpen(true)}
          className="rounded-md p-2 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          aria-label="Open menu"
        >
          <Menu className="h-5 w-5" />
        </button>
        <Link href="/dashboard" className="flex items-center gap-3">
          <NyxLogo className="h-5 w-5 text-primary" />
          <span className="text-sm font-semibold text-foreground">NYX</span>
        </Link>
        {showContextButton && onOpenContext ? (
          <button
            type="button"
            onClick={onOpenContext}
            className="ml-auto rounded-md p-2 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
            aria-label="Open context"
          >
            <PanelRightOpen className="h-5 w-5" />
          </button>
        ) : null}
      </header>

      <Drawer open={mobileOpen} onOpenChange={setMobileOpen} side="left" title="Menu">
        <div className="h-[calc(100%-52px)]">{sidebarBody}</div>
      </Drawer>

      <aside className="hidden h-dvh w-64 shrink-0 border-r border-border bg-card md:flex md:flex-col">
        {sidebarBody}
      </aside>
    </>
  );
}
