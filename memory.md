# AppTracker — decisions worth remembering

Non-obvious architectural decisions from the Cloudflare migration. These are the
"why", so future changes don't accidentally undo them.

## Neon: WebSocket driver, not HTTP

We use `drizzle-orm/neon-serverless` (the **WebSocket** `Pool` driver), *not*
`drizzle-orm/neon-http`. Reason: Better Auth uses interactive transactions
(e.g. create user + account), which the HTTP driver doesn't support. Don't
"simplify" to neon-http — it'll break auth flows.

## Local dev is offline via wsproxy

The Workers runtime can't open raw TCP, so `pg` won't run in the Worker — local
dev can't just point at a local Postgres directly. Instead `docker-compose.yml`
runs **Postgres** + a **`wsproxy`** container:

- App (Worker) → WebSocket → `wsproxy` on `localhost:4444` → Postgres. `makeDb`
  (`server/db.ts`) detects a localhost `DATABASE_URL` and switches the driver to
  the insecure local proxy (`neonConfig.wsProxy`, no TLS).
- `drizzle-kit` (migrations) → **direct TCP** to `localhost:5432`. Same one
  `DATABASE_URL` works for both because localhost resolves to the exposed PG.

Production (a `*.neon.tech` host) keeps the secure WebSocket defaults — the local
branch in `makeDb` simply doesn't trigger.

## Bindings are request-scoped

R2, `DATABASE_URL`, the Better Auth secret, etc. only exist on `c.env` per
request. So `db` and the `auth` instance are built **per request** in middleware
(`makeDb`/`makeAuth`), never as module singletons. The DB pool is closed after
the response via `c.executionCtx.waitUntil(pool.end())`.

## SPA is served by the assets binding

`wrangler.jsonc` sets `assets.not_found_handling: "single-page-application"` and
the Worker's catch-all does `c.env.ASSETS.fetch(c.req.raw)`. Static files are
served by the platform; `/api/*` falls through to Hono; unknown routes get
`index.html`.

## Tenancy = per-user

Every application row has `user_id` and every query is scoped to the session
user (set by the `/api/*` guard middleware). `goals` is keyed by `user_id` as its
primary key (one row per user), replacing the old `id = 1` singleton. Resume
isolation is enforced by the scoped query, not just the R2 key namespace.
