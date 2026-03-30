"use client";

import * as React from "react";
import { cn } from "@/lib/cn";

interface TabsContextValue {
  value: string;
  onValueChange: (value: string) => void;
  baseId: string;
}

const TabsContext = React.createContext<TabsContextValue | undefined>(
  undefined,
);

function useTabs() {
  const ctx = React.useContext(TabsContext);
  if (!ctx) throw new Error("Tabs components must be used within <Tabs>");
  return ctx;
}

interface TabsProps extends React.HTMLAttributes<HTMLDivElement> {
  value: string;
  onValueChange: (value: string) => void;
}

function Tabs({
  value,
  onValueChange,
  className,
  children,
  ...props
}: TabsProps) {
  const baseId = React.useId();
  return (
    <TabsContext.Provider value={{ value, onValueChange, baseId }}>
      <div className={cn("w-full", className)} {...props}>
        {children}
      </div>
    </TabsContext.Provider>
  );
}

function TabsList({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        "inline-flex h-9 items-center justify-center rounded-lg bg-muted p-1 text-muted-foreground",
        className,
      )}
      role="tablist"
      {...props}
    />
  );
}

interface TabsTriggerProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  value: string;
}

function TabsTrigger({
  value,
  className,
  onClick,
  onKeyDown,
  ...props
}: TabsTriggerProps) {
  const { value: selected, onValueChange, baseId } = useTabs();
  const isActive = selected === value;
  const safeValue = value.replace(/\s+/g, "-").toLowerCase();
  const tabId = `${baseId}-tab-${safeValue}`;
  const panelId = `${baseId}-panel-${safeValue}`;

  return (
    <button
      role="tab"
      type="button"
      id={tabId}
      aria-controls={panelId}
      aria-selected={isActive}
      tabIndex={isActive ? 0 : -1}
      className={cn(
        "inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1 text-sm font-medium transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
        isActive
          ? "bg-background text-foreground shadow"
          : "hover:bg-background/50 hover:text-foreground",
        className,
      )}
      onClick={(event) => {
        onValueChange(value);
        onClick?.(event);
      }}
      onKeyDown={(event) => {
        const list = event.currentTarget.closest('[role="tablist"]');
        const tabs = list
          ? Array.from(list.querySelectorAll<HTMLButtonElement>('[role="tab"]'))
          : [];
        const currentIndex = tabs.indexOf(event.currentTarget);

        if (event.key === "ArrowRight" || event.key === "ArrowDown") {
          event.preventDefault();
          const next = tabs[(currentIndex + 1 + tabs.length) % tabs.length];
          next?.focus();
          next?.click();
        } else if (event.key === "ArrowLeft" || event.key === "ArrowUp") {
          event.preventDefault();
          const prev = tabs[(currentIndex - 1 + tabs.length) % tabs.length];
          prev?.focus();
          prev?.click();
        } else if (event.key === "Home") {
          event.preventDefault();
          tabs[0]?.focus();
          tabs[0]?.click();
        } else if (event.key === "End") {
          event.preventDefault();
          const last = tabs[tabs.length - 1];
          last?.focus();
          last?.click();
        }

        onKeyDown?.(event);
      }}
      {...props}
    />
  );
}

interface TabsContentProps extends React.HTMLAttributes<HTMLDivElement> {
  value: string;
}

function TabsContent({ value, className, ...props }: TabsContentProps) {
  const { value: selected, baseId } = useTabs();
  if (selected !== value) return null;
  const safeValue = value.replace(/\s+/g, "-").toLowerCase();

  return (
    <div
      role="tabpanel"
      id={`${baseId}-panel-${safeValue}`}
      aria-labelledby={`${baseId}-tab-${safeValue}`}
      className={cn("mt-2 animate-fade-in", className)}
      {...props}
    />
  );
}

export { Tabs, TabsList, TabsTrigger, TabsContent };
