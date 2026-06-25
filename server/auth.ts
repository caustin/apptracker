import { betterAuth } from "better-auth";
import { drizzleAdapter } from "better-auth/adapters/drizzle";
import type { DB } from "./db";
import type { Env } from "./env";
import { user, session, account, verification } from "./schema";

// Built per-request from env (bindings are request-scoped on Workers). Google is
// only enabled when its credentials are present, so local email/password dev
// works without configuring an OAuth app.
export function makeAuth(env: Env, db: DB) {
  return betterAuth({
    baseURL: env.BETTER_AUTH_URL,
    secret: env.BETTER_AUTH_SECRET,
    database: drizzleAdapter(db, {
      provider: "pg",
      schema: { user, session, account, verification },
    }),
    emailAndPassword: { enabled: true },
    socialProviders:
      env.GOOGLE_CLIENT_ID && env.GOOGLE_CLIENT_SECRET
        ? {
            google: {
              clientId: env.GOOGLE_CLIENT_ID,
              clientSecret: env.GOOGLE_CLIENT_SECRET,
            },
          }
        : undefined,
  });
}

export type Auth = ReturnType<typeof makeAuth>;
