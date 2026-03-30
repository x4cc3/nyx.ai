import "@testing-library/jest-dom";

if (typeof globalThis.Response === "undefined") {
  class TestResponse {
    body: unknown;
    headers: Headers;
    ok: boolean;
    status: number;

    constructor(body?: unknown, init?: { headers?: HeadersInit; status?: number }) {
      this.body = body ?? null;
      this.status = init?.status ?? 200;
      this.headers = new Headers(init?.headers);
      this.ok = this.status >= 200 && this.status < 300;
    }

    static json(data: unknown, init?: { headers?: HeadersInit; status?: number }) {
      const headers = new Headers(init?.headers);
      if (!headers.has("content-type")) {
        headers.set("content-type", "application/json");
      }
      return new TestResponse(JSON.stringify(data), {
        ...init,
        headers,
      });
    }

    async json() {
      return JSON.parse(String(this.body ?? "null"));
    }

    async text() {
      return String(this.body ?? "");
    }
  }

  (globalThis as any).Response = TestResponse;
}
