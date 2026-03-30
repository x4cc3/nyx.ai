"use client";

export const dynamic = "force-dynamic";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import AuthSplitLayout from "@/components/auth/AuthSplitLayout";
import { Button, Input } from "@/components/ui";
import {
  SESSION_API_KEY_COOKIE,
  SESSION_TENANT_COOKIE,
  getSiteOrigin,
} from "@/lib/env";
import { createSupabaseBrowserClient } from "@/lib/supabase/browser";
import { hasSupabaseConfig } from "@/lib/supabase/config";
import { Mail } from "lucide-react";

function setCookie(name: string, value: string) {
  document.cookie = `${name}=${encodeURIComponent(value)}; Path=/; SameSite=Lax`;
}

export default function RegisterPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [tenant, setTenant] = useState("default");
  const [apiKey, setApiKey] = useState("");
  const [loading, setLoading] = useState<"signup" | "otp" | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const supabaseReady = hasSupabaseConfig();

  function persistNyxContext() {
    setCookie(SESSION_TENANT_COOKIE, tenant || "default");
    setCookie(SESSION_API_KEY_COOKIE, apiKey);
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!supabaseReady) {
      setError("Supabase auth is not configured");
      return;
    }

    setLoading("signup");
    setError(null);
    setMessage(null);
    persistNyxContext();

    const supabase = createSupabaseBrowserClient();
    const { error: signUpError, data } = await supabase.auth.signUp({
      email: email.trim(),
      password,
      options: {
        emailRedirectTo: `${getSiteOrigin()}/auth/callback?next=/dashboard`,
        data: {
          tenant_id: tenant || "default",
        },
      },
    });

    if (signUpError) {
      setError(signUpError.message);
      setLoading(null);
      return;
    }

    if (data.session) {
      router.push("/dashboard");
      return;
    }

    setMessage("Account created. Check your email to confirm sign-in.");
    setLoading(null);
  }

  async function onMagicLinkSignup() {
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
        emailRedirectTo: `${getSiteOrigin()}/auth/callback?next=/dashboard`,
        data: {
          tenant_id: tenant || "default",
        },
      },
    });

    if (otpError) {
      setError(otpError.message);
    } else {
      setMessage("Magic link sent. Open it to finish account access.");
    }
    setLoading(null);
  }

  return (
    <AuthSplitLayout
      title="Create operator account"
      subtitle="Provision a Supabase-backed identity for NYX. Tenant and backend key stay as local runtime context."
    >
      <form onSubmit={onSubmit} className="space-y-4">
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
            placeholder="create a password"
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
          {loading === "signup" ? "Creating account..." : "Create account"}
        </Button>

        <Button
          type="button"
          variant="secondary"
          className="w-full"
          onClick={() => void onMagicLinkSignup()}
          disabled={loading !== null || !email.trim()}
        >
          <Mail className="h-4 w-4" />
          {loading === "otp" ? "Sending..." : "Use magic link instead"}
        </Button>

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
        Already have an account?{" "}
        <Link className="text-primary hover:underline" href="/login">
          Sign in
        </Link>
      </div>
    </AuthSplitLayout>
  );
}
