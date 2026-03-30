import { redirect } from "next/navigation";
import { AUTH_SERVICE_UNAVAILABLE_MESSAGE } from "@/lib/env";
import { createSupabaseServerClient } from "@/lib/supabase/server";
import { hasSupabaseConfig } from "@/lib/supabase/config";

function loginRedirect(nextPath: string): never {
  const next = nextPath && nextPath.startsWith("/") ? nextPath : "/";
  redirect(`/login?next=${encodeURIComponent(next)}`);
}

export async function requireSession(nextPath: string): Promise<void> {
  if (!hasSupabaseConfig()) {
    loginRedirect(nextPath);
  }

  let user;

  try {
    const supabase = await createSupabaseServerClient();
    ({
      data: { user },
    } = await supabase.auth.getUser());
  } catch {
    throw new Error(AUTH_SERVICE_UNAVAILABLE_MESSAGE);
  }

  if (!user) {
    loginRedirect(nextPath);
  }
}
