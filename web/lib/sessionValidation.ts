export type SessionValidationState = "valid" | "invalid";

export async function validateSessionToken(
  token: string,
): Promise<SessionValidationState> {
  return token.trim() ? "valid" : "invalid";
}
