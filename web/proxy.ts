import { NextRequest, NextResponse } from "next/server";
import {
  AUTH_SERVICE_UNAVAILABLE_MESSAGE,
  SESSION_COOKIE_NAME,
} from "@/lib/env";
import { updateSupabaseSession } from "@/lib/supabase/middleware";

function isPublicPath(pathname: string) {
  if (pathname === "/") return true;
  if (pathname.startsWith("/api")) return true;
  if (pathname.startsWith("/_next")) return true;
  if (pathname === "/favicon.ico") return true;
  if (pathname.startsWith("/login")) return true;
  if (pathname.startsWith("/register")) return true;
  return false;
}

function isProtectedPath(pathname: string) {
  return pathname.startsWith("/dashboard") || pathname.startsWith("/scans");
}

export async function proxy(req: NextRequest) {
  const { pathname } = req.nextUrl;
  let response: NextResponse;
  let user: Awaited<ReturnType<typeof updateSupabaseSession>>["user"];

  try {
    ({ response, user } = await updateSupabaseSession(req));
  } catch {
    return new NextResponse(AUTH_SERVICE_UNAVAILABLE_MESSAGE, { status: 503 });
  }

  if (isPublicPath(pathname) || !isProtectedPath(pathname)) {
    return response;
  }

  if (!user) {
    const url = req.nextUrl.clone();
    url.pathname = "/login";
    url.searchParams.set("next", `${pathname}${req.nextUrl.search || ""}`);
    const redirectResponse = NextResponse.redirect(url);
    redirectResponse.cookies.delete(SESSION_COOKIE_NAME);
    return redirectResponse;
  }

  return response;
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"],
};
