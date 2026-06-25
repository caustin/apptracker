# AppTracker — notes for Claude

Multi-tenant job-search tracker: positions, people (contacts), events (emails, calls, interviews, follow-ups), uploaded PDF resumes, goals, and a dashboard with a stage funnel. Each user account is a private workspace — every application row carries a `user_id` and is scoped to the signed-in user.

## Stack

- **Runtime:** Cloudflare Workers. One Worker serves the API (Hono) and the built React app (Workers Static Assets, SPA fallback). Entry is `server/index.ts` ending in `export default app`.
- **Frontend:** Vite + React 19 + TypeScript, plain CSS (`src/styles.css`, CSS variables, light mode only). No router — tab state lives in `App.tsx`, synced to the URL hash.
- **Backend:** Hono. Drizzle ORM over **Neon Postgres** via the `@neondatabase/serverless` **WebSocket** driver (`drizzle-orm/neon-serverless`). The API is fully **async** (`await db.select()…`). Bindings are request-scoped, so `db` and the Better Auth instance are built **per request** in middleware (`makeDb`, `makeAuth`) and the pool is closed via `executionCtx.waitUntil`.
- **Auth:** Better Auth (email/password + Google) at `/api/auth/*`. A session-guard middleware on `/api/*` rejects unauthenticated requests with 401 and sets `c.get("userId")`; every query is scoped to it.
- **Database:** Neon Postgres. Schema in `server/schema.ts`; migrations via **Drizzle Kit** in `drizzle/` (`npm run db:generate` / `db:migrate`) — never create tables at runtime. Locally, Postgres + a `wsproxy` run in Docker (`docker-compose.yml`) so the same serverless driver works offline; `makeDb` switches to the insecure local proxy when `DATABASE_URL` is localhost.
- **Uploads:** resume PDFs live in **R2** (bucket binding `RESUMES`) behind the `Storage` interface in `server/storage.ts`. Object key is `<userId>/<uuid>.pdf`; the `resumes` row stores display name + that key. Served at `GET /api/resumes/:id/file` with a static `filename="resume.pdf"` + `nosniff` (never echo the user-controlled name).
- **Config/secrets:** `wrangler.jsonc` (bindings). Secrets via `.dev.vars` locally / `wrangler secret put` in prod: `DATABASE_URL`, `BETTER_AUTH_SECRET`, `BETTER_AUTH_URL`, `GOOGLE_CLIENT_ID/SECRET`.

## Commands

- `docker compose up -d` — local Postgres (`:5432`, direct TCP for drizzle-kit) + wsproxy (`:4444`, WebSocket for the app)
- `npm run dev` — Vite (`:5173`) + `wrangler dev` (`:8787`); Vite proxies `/api` to the Worker (same-origin → first-party cookies)
- `npm run db:generate` / `npm run db:migrate` — Drizzle Kit migrations
- `npm run build` — typecheck (`tsc --noEmit`) then `vite build` into `dist/`
- `npm run deploy` — build then `wrangler deploy`
- `npm run check` — typecheck only

## Layout

- `server/index.ts` — all API routes. CRUD for positions/people/events is generated from the `resources` map (Zod create/update schemas + optional `beforeSave`); every route is async and user-scoped. Resumes/goals/export are hand-written. Ends with a catch-all serving the SPA via `env.ASSETS`.
- `server/schema.ts` — Drizzle/pgTable definitions (camelCase TS ↔ snake_case columns), including Better Auth tables (`user`/`session`/`account`/`verification`) and `user_id` FKs.
- `server/db.ts`, `server/auth.ts`, `server/storage.ts`, `server/env.ts` — per-request DB factory, Better Auth config, R2 storage impl + interface, and the `Env` bindings type.
- `src/types.ts` — client-side copies of the row types plus shared constants (`STATUS_META`, `EVENT_META`) and formatters. **Kept in sync with `server/schema.ts` by hand** — update both when changing a table.
- `src/lib/auth-client.ts` — Better Auth React client (`useSession`, `signIn`, `signUp`, `signOut`).
- `src/App.tsx` — gates on `useSession()` (login screen when signed out), loads everything via `api.loadAll()` into one `Db`; mutations call the API then `refresh()`. Tabs wrapped in `ErrorBoundary`. No client-side cache or state library.
- `src/components/` — one file per tab (`Dashboard`, `Positions`, `People`, `Resumes`, `Goals`) plus `Funnel.tsx`, `Auth.tsx`, `ErrorBoundary.tsx`. `Positions.tsx` also contains the detail view, position form, and event form.

## Conventions

- Dates are ISO `YYYY-MM-DD` strings end to end; compare them lexicographically. Server `today()` uses UTC. Money is whole dollars (integers).
- Position status flow: `lead → applied → screening → interviewing → offer → accepted`, with `rejected`/`withdrawn` as terminal side-exits (`order: -1` in `STATUS_META`). The funnel counts positions whose current status is at or past each stage.
- The server stamps `appliedAt` the first time a position's status moves to `applied` or beyond (`stampApplied` beforeSave). `status`/event `type` are validated against the allowed sets server-side (Zod).
- Empty strings from forms are normalized to `NULL` (the `emptyToNull` Zod preprocessor); omitted keys are dropped so DB defaults apply.
- IDs in routes are validated (`parseId` → 400 on non-numeric); generic DELETE returns 404 when nothing was scoped/deleted.
- `goals` is one row per user (PK `user_id`), GET returns a default object when none exists, PUT upserts.
- The event type is named `Interaction` on the client to avoid clashing with the DOM `Event` type.
- `positions.resume_id` is `ON DELETE SET NULL` — deleting a resume unlinks it from positions. Resume delete removes the R2 object first (try/catch), then the row, so DB and storage stay in sync.
- Resume storage goes through the `Storage` interface so a future "bring your own cloud" backend (e.g. Google Drive) can be added without touching routes.

## Security model

- **Auth:** Better Auth `emailAndPassword` with server-enforced `minPasswordLength: 12` / `maxPasswordLength: 128`. Better Auth's built-in rate limiting is enabled everywhere (sign-in/sign-up/change-password/change-email are capped at 3 / 10s per IP; reset/verification sends at 3 / 60s). It is in-memory per-isolate (best-effort across isolates). `requireEmailVerification: false` is an **explicitly accepted risk** until an email provider is wired — flip it on before any email-based flow (reset, sharing, notifications).
- **Auth secret:** `makeAuth` fails closed in production if `BETTER_AUTH_SECRET` is missing, < 32 bytes, or equal to the `.dev.vars.example` placeholder. `BETTER_AUTH_URL` must be `https://…` in prod so cookies get the `Secure` flag and OAuth redirects are correct.
- **CSRF:** Sessions are cookies (`httpOnly`, `path: "/"`, `SameSite=Lax`, `Secure` in prod — set explicitly in `makeAuth` via `advanced.defaultCookieAttributes` / `useSecureCookies`). Defense in depth: mutating `/api/*` (POST/PUT/PATCH/DELETE) also require an `Origin`/`Referer` matching `BETTER_AUTH_URL`'s origin (`originOk` in `server/index.ts`). **Do not set `SameSite=None` or remove the Origin check without adding a real CSRF defense.** No CORS is configured, so cross-origin credentialed fetches are blocked by the browser.
- **Security headers:** `securityHeaders` middleware sets `X-Content-Type-Options`, `Referrer-Policy`, `X-Frame-Options: DENY`, `Permissions-Policy`, and a strict CSP (only on HTML responses) with `object-src 'none'` and `frame-ancestors 'none'`. The CSP allows `style-src https://fonts.googleapis.com` and `font-src https://fonts.gstatic.com` for the web fonts loaded in `index.html`. If you add inline scripts/styles or another third-party host, update `CSP` in `server/index.ts`.
- **Resume download:** served as `Content-Disposition: attachment` (never `inline`) with a static filename + `nosniff`, so a malicious PDF's embedded JS can't run in the app origin. `url`/`linkedin` are scheme-validated to http(s) server-side (Zod `httpUrl`) and again on render (`safeHttpHref` in `src/types.ts`).
- **Export DoS:** `/api/export` caps total resume bytes at `MAX_EXPORT_BYTES` (64 MB) and returns 413 early; `zipSync` materializes the archive in memory, so an unbounded set can OOM a 128 MB isolate.
- **API 404s:** unknown `/api/*` returns JSON 404 (not the SPA shell).
