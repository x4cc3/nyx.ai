"use client";

export const dynamic = "force-dynamic";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";
import AuthSplitLayout from "@/components/auth/AuthSplitLayout";
import { Button, Input } from "@/components/ui";
import {
  SESSION_API_KEY_COOKIE,
  SESSION_TENANT_COOKIE,
  getSiteOrigin,
} from "@/lib/env";
import { createSupabaseBrowserClient } from "@/lib/supabase/browser";
import { hasSupabaseConfig } from "@/lib/supabase/config";
import { ArrowRight, Github, Mail } from "lucide-react";

function setCookie(name: string, value: string) {
  document.cookie = `${name}=${encodeURIComponent(value)}; Path=/; SameSite=Lax`;
}

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [tenant, setTenant] = useState("default");
  const [apiKey, setApiKey] = useState("");
  const [loading, setLoading] = useState<"password" | "otp" | "github" | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const supabaseReady = hasSupabaseConfig();
  const nextPath = useMemo(() => {
    if (typeof window === "undefined") {
      return "/dashboard";
    }
    const rawNext = new URLSearchParams(window.location.search).get("next");
    if (!rawNext || !rawNext.startsWith("/") || rawNext.startsWith("//")) {
      return "/dashboard";
    }
    return rawNext;
  }, []);

  function persistNyxContext() {
    setCookie(SESSION_TENANT_COOKIE, tenant || "default");
    setCookie(SESSION_API_KEY_COOKIE, apiKey);
  }

  async function onPasswordSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!supabaseReady) {
      setError("Supabase auth is not configured");
      return;
    }

    setLoading("password");
    setError(null);
    setMessage(null);
    persistNyxContext();

    const supabase = createSupabaseBrowserClient();
    const { error: signInError } = await supabase.auth.signInWithPassword({
      email: email.trim(),
      password,
    });

    if (signInError) {
      setError(signInError.message);
      setLoading(null);
      return;
    }

    router.push(nextPath);
  }

  async function onMagicLink() {
    if (!supabaseReady) {
      setError("Supabase auth is not configured");
      return;
    }

    setLoading("otp");
    setError(null);
    setMessage(null);
    persistNyxContext();

    const supabase = createSupabaseBrowserClient();
    const { error: otpError } = await supabase.auth.signInWithOtp({
      email: email.trim(),
      options: {
        emailRedirectTo: `${getSiteOrigin()}/auth/callback?next=${encodeURIComponent(nextPath)}`,
      },
    });

    if (otpError) {
      setError(otpError.message);
    } else {
      setMessage("Magic link sent. Check your inbox.");
    }
    setLoading(null);
  }

  async function onGithub() {
    if (!supabaseReady) {
      setError("Supabase auth is not configured");
      return;
    }

    setLoading("github");
    setError(null);
    setMessage(null);
    persistNyxContext();

    const supabase = createSupabaseBrowserClient();
    const { error: oauthError } = await supabase.auth.signInWithOAuth({
      provider: "github",
      options: {
        redirectTo: `${getSiteOrigin()}/auth/callback?next=${encodeURIComponent(nextPath)}`,
      },
    });

    if (oauthError) {
      setError(oauthError.message);
      setLoading(null);
    }
  }

  return (
    <AuthSplitLayout
      title="Operator access"
      subtitle="Supabase handles sign-in and session refresh. NYX keeps tenant and backend context alongside that identity."
    >
      <form onSubmit={onPasswordSubmit} className="space-y-4">
        <div className="space-y-1">
          <label htmlFor="email" className="text-sm font-medium text-foreground">
            Email
          </label>
          <Input
            id="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="operator@nyx.ai"
            className="font-mono border-border bg-background"
            type="email"
            required
          />
        </div>

        <div className="space-y-1">
          <label
            htmlFor="password"
            className="text-sm font-medium text-foreground"
          >
            Password
          </label>
          <Input
            id="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="••••••••••••"
            className="font-mono border-border bg-background"
            type="password"
            required
          />
        </div>

        <div className="grid gap-4 sm:grid-cols-2">
          <div className="space-y-1">
            <label
              htmlFor="tenant"
              className="text-sm font-medium text-foreground"
            >
              Tenant
            </label>
            <Input
              id="tenant"
              value={tenant}
              onChange={(e) => setTenant(e.target.value)}
              placeholder="default"
              className="font-mono border-border bg-background"
            />
          </div>

          <div className="space-y-1">
            <label
              htmlFor="api-key"
              className="text-sm font-medium text-foreground"
            >
              NYX API key
            </label>
            <Input
              id="api-key"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              placeholder="optional backend key"
              className="font-mono border-border bg-background"
              type="password"
            />
          </div>
        </div>

        {error ? (
          <div className="rounded-md border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-200">
            {error}
          </div>
        ) : null}

        {message ? (
          <div className="rounded-md border border-primary/30 bg-primary/10 px-3 py-2 text-sm text-foreground">
            {message}
          </div>
        ) : null}

        <Button type="submit" className="w-full" disabled={loading !== null}>
          <ArrowRight className="h-4 w-4" />
          {loading === "password" ? "Signing in..." : "Sign in"}
        </Button>

        <div className="grid gap-2 sm:grid-cols-2">
          <Button
            type="button"
            variant="secondary"
            className="w-full"
            onClick={() => void onMagicLink()}
            disabled={loading !== null || !email.trim()}
          >
            <Mail className="h-4 w-4" />
            {loading === "otp" ? "Sending..." : "Magic link"}
          </Button>
          <Button
            type="button"
            variant="outline"
            className="w-full"
            onClick={() => void onGithub()}
            disabled={loading !== null}
          >
            <Github className="h-4 w-4" />
            {loading === "github" ? "Redirecting..." : "GitHub"}
          </Button>
        </div>

        {!supabaseReady ? (
          <div className="rounded-md border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-sm leading-6 text-amber-200">
            Set{" "}
            <span className="break-all font-mono">NEXT_PUBLIC_SUPABASE_URL</span>{" "}
            and{" "}
            <span className="break-all font-mono">
              NEXT_PUBLIC_SUPABASE_ANON_KEY
            </span>{" "}
            to enable auth.
          </div>
        ) : null}
      </form>

      <div className="text-sm text-muted-foreground">
        Need a new operator account?{" "}
        <Link className="text-primary hover:underline" href="/register">
          Create one
        </Link>
      </div>
    </AuthSplitLayout>
  );
}
