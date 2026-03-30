import { NextRequest } from "next/server";
import {
  API_BASE_URL,
  API_KEY,
  SESSION_API_KEY_COOKIE,
  SESSION_TENANT_COOKIE,
} from "@/lib/env";
import { createSupabaseServerClient } from "@/lib/supabase/server";

function buildBackendUrl(req: NextRequest, pathSegments: string[]) {
  const safePath = pathSegments.join("/").replace(/^\/+/, "");
  const url = new URL(`${API_BASE_URL}/${safePath}`);
  url.search = new URL(req.url).search;
  return url;
}

function cookieValue(req: NextRequest, name: string): string {
  return req.cookies.get(name)?.value?.trim() ?? "";
}

async function sanitizeHeaders(req: NextRequest) {
  const headers = new Headers(req.headers);
  headers.delete("host");
  headers.delete("connection");
  headers.delete("content-length");
  headers.delete("accept-encoding");
  headers.delete("x-forwarded-for");
  headers.delete("x-real-ip");
  headers.delete("x-nyx-client-ip");
  headers.delete("authorization");

  const tenant = cookieValue(req, SESSION_TENANT_COOKIE) || "default";
  const apiKey = cookieValue(req, SESSION_API_KEY_COOKIE) || API_KEY;
  let operator = "operator";

  try {
    const supabase = await createSupabaseServerClient();
    const {
      data: { session },
      error,
    } = await supabase.auth.getSession();
    if (!error && session?.access_token) {
      headers.set("Authorization", `Bearer ${session.access_token}`);
    }
    if (session?.user?.email) {
      operator = session.user.email;
    }
  } catch {
    headers.delete("authorization");
  }

  headers.set("X-NYX-Tenant", tenant);
  headers.set("X-NYX-Operator", operator);
  if (apiKey) {
    headers.set("X-NYX-API-Key", apiKey);
  } else {
    headers.delete("X-NYX-API-Key");
  }

  return headers;
}

async function proxyRequest(req: NextRequest, pathSegments: string[]) {
  const safeSegments = pathSegments.filter(Boolean);
  if (safeSegments.some((segment) => segment === "." || segment === "..")) {
    return Response.json(
      {
        error: {
          code: "invalid_path",
          message: "Invalid API path",
        },
      },
      { status: 400 },
    );
  }

  const url = buildBackendUrl(req, safeSegments);
  const headers = await sanitizeHeaders(req);
  const method = req.method.toUpperCase();
  const body =
    method === "GET" || method === "HEAD" ? undefined : await req.text();

  try {
    const upstream = await fetch(url.toString(), {
      method,
      headers,
      body: body && body.length > 0 ? body : undefined,
      cache: "no-store",
    });

    const responseHeaders = new Headers(upstream.headers);
    responseHeaders.delete("content-encoding");
    responseHeaders.delete("content-length");
    responseHeaders.delete("transfer-encoding");

    return new Response(upstream.body, {
      status: upstream.status,
      headers: responseHeaders,
    });
  } catch {
    return Response.json(
      {
        error: {
          code: "upstream_unreachable",
          message: "Unable to reach backend API",
        },
      },
      { status: 502 },
    );
  }
}

export const runtime = "nodejs";

type RouteContext = { params: Promise<{ path?: string[] }> };

async function resolvePathSegments(context: RouteContext) {
  const { path } = await context.params;
  return path ?? [];
}

export async function GET(req: NextRequest, context: RouteContext) {
  return proxyRequest(req, await resolvePathSegments(context));
}

export async function POST(req: NextRequest, context: RouteContext) {
  return proxyRequest(req, await resolvePathSegments(context));
}

export async function PUT(req: NextRequest, context: RouteContext) {
  return proxyRequest(req, await resolvePathSegments(context));
}

export async function PATCH(req: NextRequest, context: RouteContext) {
  return proxyRequest(req, await resolvePathSegments(context));
}

export async function DELETE(req: NextRequest, context: RouteContext) {
  return proxyRequest(req, await resolvePathSegments(context));
}
