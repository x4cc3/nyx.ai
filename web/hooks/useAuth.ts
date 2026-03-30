// useAuth wraps useCurrentUser for convenience.
// useCurrentUser already manages Supabase auth state, session changes, and logout.
export { useCurrentUser as useAuth } from "./useCurrentUser";
