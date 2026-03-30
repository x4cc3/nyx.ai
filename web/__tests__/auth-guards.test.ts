const nextMock = jest.fn(() => ({ kind: "next" }));
const redirectMock = jest.fn(
  (url: { pathname?: string; searchParams?: URLSearchParams }) => ({
    kind: "redirect",
    url,
    cookies: { delete: jest.fn() },
  }),
);
const updateSupabaseSessionMock = jest.fn();
const createSupabaseServerClientMock = jest.fn();
const hasSupabaseConfigMock = jest.fn(() => true);

class MockNextResponse {
  body: string;
  status: number;
  cookies: { delete: jest.Mock };

  constructor(body?: string, init?: { status?: number }) {
    this.body = body ?? "";
    this.status = init?.status ?? 200;
    this.cookies = { delete: jest.fn() };
  }

  static next = nextMock;
  static redirect = redirectMock;
}
const navigationRedirectMock = jest.fn((url: string) => {
  throw new Error(`redirect:${url}`);
});

jest.mock("next/server", () => ({
  NextResponse: MockNextResponse,
}));

jest.mock("next/navigation", () => ({
  redirect: navigationRedirectMock,
}));

jest.mock("@/lib/env", () => ({
  API_BASE_URL: "http://api.test",
  API_KEY: "test-api-key",
  AUTH_SERVICE_UNAVAILABLE_MESSAGE:
    "Authentication service temporarily unavailable",
  MISSING_API_KEY_MESSAGE: "missing api key",
  SESSION_COOKIE_NAME: "nyx_session",
}));

jest.mock("@/lib/supabase/middleware", () => ({
  updateSupabaseSession: updateSupabaseSessionMock,
}));

jest.mock("@/lib/supabase/server", () => ({
  createSupabaseServerClient: createSupabaseServerClientMock,
}));

jest.mock("@/lib/supabase/config", () => ({
  hasSupabaseConfig: hasSupabaseConfigMock,
}));

import { proxy } from "@/proxy";
import { requireSession } from "@/lib/serverAuth";

function makeProtectedRequest(token = "session-token") {
  const clonedUrl = {
    pathname: "/dashboard",
    search: "",
    searchParams: new URLSearchParams(),
  };
  return {
    nextUrl: {
      pathname: "/dashboard",
      search: "",
      clone: () => clonedUrl,
    },
    headers: {
      get: jest.fn(() => null),
    },
    cookies: {
      get: jest.fn(() => (token ? { value: token } : undefined)),
    },
  };
}

describe("auth guards", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    hasSupabaseConfigMock.mockReturnValue(true);
  });

  it("fails closed without clearing the session cookie when proxy auth validation hits a fetch error", async () => {
    updateSupabaseSessionMock.mockRejectedValue(new Error("backend down"));

    const result = await proxy(makeProtectedRequest() as never);

    expect(result.status).toBe(503);
    expect(redirectMock).not.toHaveBeenCalled();
  });

  it("still redirects and clears the session cookie for an invalid session", async () => {
    updateSupabaseSessionMock.mockResolvedValue({
      response: MockNextResponse.next(),
      user: null,
    });

    const result = await proxy(makeProtectedRequest() as never);

    expect(redirectMock).toHaveBeenCalledTimes(1);
    expect(result.kind).toBe("redirect");
    expect(result.cookies.delete).toHaveBeenCalledWith("nyx_session");
  });

  it("fails closed server-side on transient auth fetch failures without redirecting to login", async () => {
    createSupabaseServerClientMock.mockResolvedValue({
      auth: {
        getUser: jest.fn().mockRejectedValue(new Error("backend down")),
      },
    });

    await expect(requireSession("/dashboard")).rejects.toThrow(
      "Authentication service temporarily unavailable",
    );
    expect(navigationRedirectMock).not.toHaveBeenCalled();
  });

  it("still redirects server-side when the backend confirms the session is invalid", async () => {
    createSupabaseServerClientMock.mockResolvedValue({
      auth: {
        getUser: jest.fn().mockResolvedValue({
          data: { user: null },
        }),
      },
    });

    await expect(requireSession("/dashboard")).rejects.toThrow(
      "redirect:/login?next=%2Fdashboard",
    );
  });
});
