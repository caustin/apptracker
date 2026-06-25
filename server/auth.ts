import { betterAuth } from "better-auth";
import { drizzleAdapter } from "better-auth/adapters/drizzle";
import type { DB } from "./db";
import type { Env } from "./env";
import { user, session, account, verification } from "./schema";

// Known-bad placeholder shipped in .dev.vars.example. Rejecting it in
// production prevents an operator copying the example into a secret without
// rotating it (which would expose the session signing key).
const KNOWN_PLACEHOLDER_SECRET =
  "dev-only-secret-change-me-0000000000000000";
const LOCAL_AUTH_HOSTS = new Set([
  "localhost",
  "127.0.0.1",
  "0.0.0.0",
  "::1",
  "[::1]",
]);

function isLocalAuthHost(hostname: string) {
  return LOCAL_AUTH_HOSTS.has(hostname) || hostname.endsWith(".localhost");
}

// Workers bindings only exist per-request, so the auth instance is built
// per-request from env rather than as a module singleton. Google is only
// enabled when its credentials are present, so local email/password dev
// works without configuring an OAuth app.
export function makeAuth(env: Env, db: DB, request?: Request) {
  // BETTER_AUTH_URL drives cookie Secure/attributes and OAuth redirects. An
  // empty/garbage value silently degrades security (no Secure flag, CSRF
  // origin check disabled), so fail closed on it regardless of environment.
  let baseURL: URL;
  try {
    baseURL = new URL(env.BETTER_AUTH_URL);
  } catch {
    throw new Error("BETTER_AUTH_URL is missing or not a valid URL");
  }
  if (baseURL.protocol !== "http:" && baseURL.protocol !== "https:") {
    throw new Error("BETTER_AUTH_URL must be an http(s) URL");
  }
  const isLocalDev = isLocalAuthHost(baseURL.hostname);
  const requestURL = request ? new URL(request.url) : null;
  const isLocalRequest = requestURL
    ? isLocalAuthHost(requestURL.hostname)
    : isLocalDev;
  const isProd = !isLocalRequest;
  if (isProd && baseURL.protocol !== "https:") {
    throw new Error("BETTER_AUTH_URL must be https:// in production");
  }
  if (isProd && isLocalDev) {
    throw new Error("BETTER_AUTH_URL must not point to localhost in production");
  }
  if (isProd && requestURL && baseURL.origin !== requestURL.origin) {
    throw new Error(
      "BETTER_AUTH_URL must match the deployed app origin in production",
    );
  }
  // Fail closed on a missing/placeholder session secret in production. In
  // dev any non-empty value is allowed so local setup still works.
  if (isProd) {
    const s = env.BETTER_AUTH_SECRET;
    if (!s || s.length < 32 || s === KNOWN_PLACEHOLDER_SECRET) {
      throw new Error("BETTER_AUTH_SECRET is missing or uses a placeholder");
    }
  }

  return betterAuth({
    baseURL: env.BETTER_AUTH_URL,
    secret: env.BETTER_AUTH_SECRET,
    database: drizzleAdapter(db, {
      provider: "pg",
      schema: { user, session, account, verification },
    }),
    emailAndPassword: {
      enabled: true,
      // Enforced server-side (Zod in Better Auth's sign-up route); the
      // client-side minLength is just a UX hint and trivially bypassed.
      minPasswordLength: 12,
      maxPasswordLength: 128,
      // Email verification requires an email provider, which is not yet
      // wired (see TODO.md). Accepted risk: email is only a login
      // identifier today; before any email-based flow (reset, sharing,
      // notifications) lands, verification must be required.
      requireEmailVerification: false,
    },
    // Rate limiting is enabled by Better Auth in production, but we enable it
    // in dev too. Special rules apply to auth endpoints: sign-in / sign-up /
    // change-password / change-email are capped at 3 requests / 10s per IP,
    // reset/verification sends at 3 / 60s. Storage is in-memory (per-isolate)
    // so it is best-effort across isolates, but it raises the bar for
    // brute-force / credential-stuffing and account-spam.
    rateLimit: { enabled: true },
    advanced: {
      ipAddress: {
        ipAddressHeaders: ["cf-connecting-ip", "x-forwarded-for"],
      },
      // Force Secure cookies in production (Better Auth would infer this from
      // the https baseURL, but make it explicit so a misconfigured URL can't
      // silently drop the Secure flag). httpOnly + path "/" are Better Auth
      // defaults; SameSite is restated explicitly so a future library default
      // change can't quietly weaken the CSRF posture (see CLAUDE.md).
      useSecureCookies: isProd,
      defaultCookieAttributes: {
        httpOnly: true,
        sameSite: "lax",
        path: "/",
        secure: isProd,
      },
    },
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
