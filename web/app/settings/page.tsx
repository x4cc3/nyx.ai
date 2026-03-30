"use client";

export const dynamic = "force-dynamic";

import { useEffect, useState } from "react";
import AppShell from "@/components/layout/AppShell";
import { Button, Card, CardContent, Input } from "@/components/ui";

const SESSION_TENANT_COOKIE = "nyx_tenant";
const SESSION_API_KEY_COOKIE = "nyx_api_key";
const LOCAL_PREFS_KEY = "nyx.operator.preferences";

type OperatorPreferences = {
  searchProvider: string;
  executorMode: string;
};

function readCookie(name: string): string {
  const match = document.cookie.match(
    new RegExp(`(?:^|; )${name.replace(/[$()*+.?[\\\]^{|}]/g, "\\$&")}=([^;]*)`),
  );
  return match ? decodeURIComponent(match[1]) : "";
}

function writeCookie(name: string, value: string) {
  const encoded = encodeURIComponent(value.trim());
  document.cookie = `${name}=${encoded}; path=/; max-age=${60 * 60 * 24 * 30}; samesite=lax`;
}

export default function SettingsPage() {
  const [tenant, setTenant] = useState("default");
  const [apiKey, setApiKey] = useState("");
  const [searchProvider, setSearchProvider] = useState("auto");
  const [executorMode, setExecutorMode] = useState("auto");
  const [savedMessage, setSavedMessage] = useState<string | null>(null);

  useEffect(() => {
    setTenant(readCookie(SESSION_TENANT_COOKIE) || "default");
    setApiKey(readCookie(SESSION_API_KEY_COOKIE));

    const rawPrefs = window.localStorage.getItem(LOCAL_PREFS_KEY);
    if (!rawPrefs) return;
    try {
      const prefs = JSON.parse(rawPrefs) as Partial<OperatorPreferences>;
      if (prefs.searchProvider) setSearchProvider(prefs.searchProvider);
      if (prefs.executorMode) setExecutorMode(prefs.executorMode);
    } catch {
      window.localStorage.removeItem(LOCAL_PREFS_KEY);
    }
  }, []);

  function handleSave() {
    writeCookie(SESSION_TENANT_COOKIE, tenant || "default");
    writeCookie(SESSION_API_KEY_COOKIE, apiKey);
    window.localStorage.setItem(
      LOCAL_PREFS_KEY,
      JSON.stringify({ searchProvider, executorMode } satisfies OperatorPreferences),
    );
    setSavedMessage("Settings saved for this browser session.");
    setTimeout(() => setSavedMessage(null), 4000);
  }

  return (
    <AppShell>
      <div className="space-y-6 px-4 py-4 md:px-6 md:py-6 xl:px-8">
        <div className="space-y-2">
          <h1 className="text-2xl font-semibold text-foreground">Settings</h1>
          <p className="max-w-2xl text-sm leading-6 text-muted-foreground">
            Store operator defaults for tenant routing, API access, and local UI preferences.
          </p>
        </div>

        <div className="grid gap-4 xl:grid-cols-[minmax(0,1.2fr)_320px]">
          <Card className="border-border bg-card">
            <CardContent className="space-y-5 p-5">
              <div className="space-y-2">
                <label className="text-sm font-medium text-foreground">
                  Tenant header
                </label>
                <Input
                  value={tenant}
                  onChange={(event) => setTenant(event.target.value)}
                  placeholder="default"
                  className="font-mono"
                />
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium text-foreground">
                  Session API key
                </label>
                <Input
                  type="password"
                  value={apiKey}
                  onChange={(event) => setApiKey(event.target.value)}
                  placeholder="Paste NYX API key"
                  className="font-mono"
                />
              </div>

              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <label className="text-sm font-medium text-foreground">
                    Preferred search provider
                  </label>
                  <select
                    value={searchProvider}
                    onChange={(event) => setSearchProvider(event.target.value)}
                    className="h-10 w-full rounded-md border border-border bg-background px-3 text-sm text-foreground"
                  >
                    <option value="auto">Auto</option>
                    <option value="duckduckgo">DuckDuckGo</option>
                    <option value="searxng">SearxNG</option>
                    <option value="tavily">Tavily</option>
                    <option value="perplexity">Perplexity</option>
                    <option value="sploitus">Sploitus</option>
                  </select>
                </div>

                <div className="space-y-2">
                  <label className="text-sm font-medium text-foreground">
                    Preferred executor mode
                  </label>
                  <select
                    value={executorMode}
                    onChange={(event) => setExecutorMode(event.target.value)}
                    className="h-10 w-full rounded-md border border-border bg-background px-3 text-sm text-foreground"
                  >
                    <option value="auto">Auto</option>
                    <option value="docker">Docker</option>
                    <option value="local">Local</option>
                  </select>
                </div>
              </div>

              {savedMessage ? (
                <div className="rounded-md border border-border bg-background px-4 py-3 text-sm text-muted-foreground">
                  {savedMessage}
                </div>
              ) : null}

              <Button onClick={handleSave}>Save settings</Button>
            </CardContent>
          </Card>

          <Card className="border-border bg-card">
            <CardContent className="space-y-4 p-5">
              <h2 className="text-base font-semibold text-foreground">
                What this controls
              </h2>
              <p className="text-sm leading-6 text-muted-foreground">
                Tenant and API key values are stored in browser cookies so the Next proxy can forward them to the Go API.
              </p>
              <p className="text-sm leading-6 text-muted-foreground">
                Search-provider and executor preferences are saved locally and meant for operator defaults until server-side settings land.
              </p>
            </CardContent>
          </Card>
        </div>
      </div>
    </AppShell>
  );
}
