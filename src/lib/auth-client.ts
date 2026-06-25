import { createAuthClient } from "better-auth/react";

// Same-origin: the client hits /api/auth/* on the current origin (proxied to the
// Worker in dev), so cookies are first-party.
export const authClient = createAuthClient();

export const { useSession, signIn, signUp, signOut } = authClient;
