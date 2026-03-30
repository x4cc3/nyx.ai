"use client";

import Link from "next/link";
import type { ReactNode } from "react";
import { NyxLogo } from "@/components/ui/NyxLogo";

type AuthSplitLayoutProps = {
  title: ReactNode;
  subtitle: ReactNode;
  children: ReactNode;
};

export default function AuthSplitLayout({
  title,
  subtitle,
  children,
}: AuthSplitLayoutProps) {
  return (
    <div className="min-h-dvh bg-transparent text-foreground">
      <header className="border-b border-border">
        <div className="mx-auto flex w-full max-w-5xl items-center justify-between px-4 py-4 md:px-6">
          <Link href="/" className="flex items-center gap-3">
            <NyxLogo className="h-6 w-6 text-primary" />
            <span className="text-sm font-semibold text-foreground">NYX</span>
          </Link>
          <Link
            href="/"
            className="text-sm text-muted-foreground transition-colors hover:text-foreground"
          >
            Home
          </Link>
        </div>
      </header>

      <main className="mx-auto flex min-h-[calc(100dvh-65px)] w-full max-w-md items-center px-4 py-10">
        <div className="w-full rounded-lg border border-border bg-card">
          <div className="space-y-6 p-5 sm:p-6">
            <div className="space-y-2">
              <h1 className="text-2xl font-semibold text-foreground">{title}</h1>
              <p className="text-sm leading-6 text-muted-foreground">{subtitle}</p>
            </div>

            {children}
          </div>
        </div>
      </main>
    </div>
  );
}
