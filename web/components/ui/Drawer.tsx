"use client";

import * as React from "react";
import { cn } from "@/lib/cn";

type DrawerSide = "left" | "right";

export interface DrawerProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  side?: DrawerSide;
  title?: string;
  children: React.ReactNode;
}

export default function Drawer({
  open,
  onOpenChange,
  side = "right",
  title,
  children,
}: DrawerProps) {
  const panelRef = React.useRef<HTMLDivElement | null>(null);
  const restoreFocusRef = React.useRef<HTMLElement | null>(null);

  React.useEffect(() => {
    if (!open) {
      return;
    }

    restoreFocusRef.current = document.activeElement as HTMLElement | null;
    const prevOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";

    const getFocusable = () => {
      const panel = panelRef.current;
      if (!panel) return [] as HTMLElement[];
      return Array.from(
        panel.querySelectorAll<HTMLElement>(
          'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])',
        ),
      ).filter((node) => !node.hasAttribute("disabled"));
    };

    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onOpenChange(false);
        return;
      }

      if (e.key !== "Tab") {
        return;
      }

      const focusable = getFocusable();
      if (focusable.length === 0) {
        e.preventDefault();
        panelRef.current?.focus();
        return;
      }

      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      const active = document.activeElement as HTMLElement | null;

      if (e.shiftKey) {
        if (
          !active ||
          active === first ||
          !panelRef.current?.contains(active)
        ) {
          e.preventDefault();
          last.focus();
        }
        return;
      }

      if (!active || active === last || !panelRef.current?.contains(active)) {
        e.preventDefault();
        first.focus();
      }
    };

    window.addEventListener("keydown", onKeyDown);
    window.requestAnimationFrame(() => {
      const focusable = getFocusable();
      if (focusable[0]) {
        focusable[0].focus();
      } else {
        panelRef.current?.focus();
      }
    });

    return () => {
      window.removeEventListener("keydown", onKeyDown);
      document.body.style.overflow = prevOverflow;
      restoreFocusRef.current?.focus();
      restoreFocusRef.current = null;
    };
  }, [open, onOpenChange]);

  if (!open) {
    return null;
  }

  const isLeft = side === "left";

  return (
    <div className="fixed inset-0 z-50">
      <button
        type="button"
        aria-label="Close"
        className="absolute inset-0 bg-background/70 backdrop-blur-[2px]"
        onClick={() => onOpenChange(false)}
      />
      <div
        ref={panelRef}
        tabIndex={-1}
        role="dialog"
        aria-modal="true"
        aria-label={title || "Drawer"}
        className={cn(
          [
            "absolute top-0 flex h-full w-[min(420px,92vw)] flex-col",
            "border border-border bg-card shadow-2xl shadow-black/60",
            "outline-none",
            "motion-safe:animate-fade-in",
          ].join(" "),
          isLeft ? "left-0 border-l-0" : "right-0 border-r-0",
        )}
      >
        {title ? (
          <div className="flex items-center justify-between border-b border-border px-4 py-3">
            <div className="text-xs font-semibold tracking-widest text-muted-foreground uppercase">
              {title}
            </div>
            <button
              type="button"
              className="rounded-lg px-2 py-1 text-xs text-muted-foreground hover:bg-muted/25 hover:text-foreground"
              onClick={() => onOpenChange(false)}
            >
              Close
            </button>
          </div>
        ) : null}
        <div className="min-h-0 flex-1 overflow-y-auto">{children}</div>
      </div>
    </div>
  );
}
