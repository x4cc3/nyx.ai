import { act, renderHook, waitFor } from "@testing-library/react";

const signOutMock = jest.fn();
const getUserMock = jest.fn();
const onAuthStateChangeMock = jest.fn();
const createSupabaseBrowserClientMock = jest.fn();
const hasSupabaseConfigMock = jest.fn(() => true);

jest.mock("@/lib/supabase/browser", () => ({
  createSupabaseBrowserClient: createSupabaseBrowserClientMock,
}));

jest.mock("@/lib/supabase/config", () => ({
  hasSupabaseConfig: hasSupabaseConfigMock,
}));

import { useCurrentUser } from "@/hooks/useCurrentUser";

describe("useCurrentUser", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    hasSupabaseConfigMock.mockReturnValue(true);
    getUserMock.mockResolvedValue({
      data: { user: { email: "user@example.com" } },
    });
    signOutMock.mockResolvedValue({
      error: { message: "temporary failure" },
    });
    onAuthStateChangeMock.mockReturnValue({
      data: {
        subscription: {
          unsubscribe: jest.fn(),
        },
      },
    });
    createSupabaseBrowserClientMock.mockReturnValue({
      auth: {
        getUser: getUserMock,
        onAuthStateChange: onAuthStateChangeMock,
        signOut: signOutMock,
      },
    });
  });

  it("preserves the current user when logout fails transiently", async () => {
    const { result } = renderHook(() => useCurrentUser());

    await waitFor(() => expect(result.current.email).toBe("user@example.com"));

    await act(async () => {
      await expect(result.current.logout()).resolves.toBe(false);
    });

    expect(result.current.email).toBe("user@example.com");
  });
});
