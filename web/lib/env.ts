const DEFAULT_API_BASE = "http://localhost:8080/api";
const DEFAULT_SESSION_COOKIE = "nyx_session";
const DEFAULT_SITE_URL = "http://localhost:3000";

function getEnv(name: keyof NodeJS.ProcessEnv): string | undefined {
  const value = process.env[name];
  return typeof value === "string" && value.length > 0 ? value : undefined;
}

export const SESSION_COOKIE_NAME =
  getEnv("NYX_SESSION_COOKIE_NAME") ?? DEFAULT_SESSION_COOKIE;
export const SESSION_TENANT_COOKIE = "nyx_tenant";
export const SESSION_API_KEY_COOKIE = "nyx_api_key";

export const API_BASE_URL = (
  getEnv("NYX_API_BASE_URL") ?? DEFAULT_API_BASE
).replace(/\/$/, "");
export const API_KEY = getEnv("NYX_API_KEY") ?? "";
export const MISSING_API_KEY_MESSAGE =
  "NYX API key is required for this environment";
export const AUTH_SERVICE_UNAVAILABLE_MESSAGE =
  "Authentication service temporarily unavailable";

export function getSiteOrigin(): string {
  const raw =
    getEnv("NYX_SITE_URL") ??
    getEnv("NEXT_PUBLIC_SITE_URL") ??
    DEFAULT_SITE_URL;
  try {
    return new URL(raw).origin;
  } catch {
    return DEFAULT_SITE_URL;
  }
}

export function getMetadataBase(): URL {
  return new URL(getSiteOrigin());
}
