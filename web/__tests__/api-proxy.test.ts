jest.mock("@/lib/env", () => ({
  API_BASE_URL: "http://api.test/api",
  API_KEY: "test-api-key",
  MISSING_API_KEY_MESSAGE: "missing api key",
  SESSION_API_KEY_COOKIE: "nyx_api_key",
  SESSION_TENANT_COOKIE: "nyx_tenant",
}));

jest.mock("@/lib/supabase/server", () => ({
  createSupabaseServerClient: jest.fn().mockRejectedValue(
    new Error("supabase unavailable"),
  ),
}));

import { GET } from "@/app/api/[...path]/route";

describe("api proxy header sanitization", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    global.fetch = jest.fn().mockResolvedValue(
      new Response("ok", {
        status: 200,
        headers: { "content-type": "text/plain" },
      }),
    ) as jest.Mock;
  });

  it("drops public client IP headers before proxying to the backend", async () => {
    const req = {
      url: "https://example.com/api/scans?limit=1",
      method: "GET",
      headers: new Headers({
        "x-real-ip": "203.0.113.9",
        "x-forwarded-for": "198.51.100.7",
      }),
      cookies: {
        get: jest.fn(() => undefined),
      },
    };

    await GET(req as never, {
      params: Promise.resolve({ path: ["scans"] }),
    } as never);

    expect(global.fetch).toHaveBeenCalledTimes(1);
    const [url, init] = (global.fetch as jest.Mock).mock.calls[0] as [
      string,
      RequestInit,
    ];
    expect(url).toBe("http://api.test/api/scans?limit=1");

    const headers = new Headers(init.headers);
    expect(headers.get("x-nyx-client-ip")).toBeNull();
    expect(headers.get("x-forwarded-for")).toBeNull();
    expect(headers.get("x-real-ip")).toBeNull();
    expect(headers.get("authorization")).toBeNull();
    expect(headers.get("x-nyx-api-key")).toBe("test-api-key");
    expect(headers.get("x-nyx-tenant")).toBe("default");
    expect(headers.get("x-nyx-operator")).toBe("operator");
  });
});
