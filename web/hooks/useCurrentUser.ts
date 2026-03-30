"use client";

import { useCallback, useEffect, useState } from "react";
import { SESSION_API_KEY_COOKIE, SESSION_TENANT_COOKIE } from "@/lib/env";
import { createSupabaseBrowserClient } from "@/lib/supabase/browser";
import { hasSupabaseConfig } from "@/lib/supabase/config";

function readCookie(name: string): string | null {
  const encoded = encodeURIComponent(name);
  const match = document.cookie.match(new RegExp(`(?:^|; )${encoded}=([^;]*)`));
  return match ? decodeURIComponent(match[1]) : null;
}

function clearCookie(name: string) {
  document.cookie = `${name}=; Max-Age=0; Path=/; SameSite=Lax`;
}

export function useCurrentUser() {
  const [email, setEmail] = useState<string | null>(null);

  useEffect(() => {
    if (!hasSupabaseConfig()) {
      return;
    }

    let active = true;
    const supabase = createSupabaseBrowserClient();

    void supabase.auth.getUser()
      .then(({ data }) => {
        if (!active) return;
        setEmail(data.user?.email ?? null);
      })
      .catch(() => {
        if (active) setEmail(null);
      });

    const {
      data: { subscription },
    } = supabase.auth.onAuthStateChange((_event, session) => {
      if (!active) return;
      setEmail(session?.user?.email ?? null);
    });

    return () => {
      active = false;
      subscription.unsubscribe();
    };
  }, []);

  const clearCurrentUser = useCallback(() => {
    setEmail(null);
  }, []);

  const logout = useCallback(async (): Promise<boolean> => {
    if (hasSupabaseConfig()) {
      const supabase = createSupabaseBrowserClient();
      const { error } = await supabase.auth.signOut();
      if (error) {
        return false;
      }
    }
    clearCookie(SESSION_TENANT_COOKIE);
    clearCookie(SESSION_API_KEY_COOKIE);
    setEmail(null);
    return true;
  }, []);

  return {
    email,
    clearCurrentUser,
    logout,
  };
}
