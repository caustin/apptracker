# AppTracker — open items

Tracking work left after the Cloudflare Workers / multi-tenant migration.

## Before production launch

- [ ] **Create Neon project** and grab its connection string.
- [ ] **Create R2 bucket** named `apptracker-resumes` (matches `wrangler.jsonc`).
- [ ] **Set production secrets** (`wrangler secret put …`): `DATABASE_URL`,
      `BETTER_AUTH_SECRET` (`openssl rand -base64 32`), `BETTER_AUTH_URL`
      (`https://your-domain`), `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`.
- [ ] **Migrate the prod DB**: `DATABASE_URL="<neon-url>" npm run db:migrate`.
- [ ] **Deploy**: `npm run deploy`, then smoke-test the live URL end to end.

## Auth

- [ ] **Google OAuth** — create an OAuth client. Redirects:
      `http://localhost:5173/api/auth/callback/google` (local) and
      `https://your-domain/api/auth/callback/google` (prod). Until then, only
      email/password works. The "Continue with Google" button errors without it.
- [ ] **Email verification / password reset** — Better Auth stores
      `emailVerified` but no mailer is configured, so verification + reset emails
      aren't sent. Wire up an email provider if you want these flows.

## Data

- [ ] **Migrate existing local data** — the old `data/apptracker.db` is *not*
      carried over. If you want those rows in Neon, write a one-off
      export/import that assigns them to your first account.

## Nice-to-have / later

- [ ] **Hyperdrive** in front of Neon if session-lookup latency becomes
      noticeable (adds edge pooling/caching).
- [ ] **Resume virus scanning** — currently skipped (PDF-only, each user only
      downloads their own uploads). Revisit with a quarantine-scan pipeline if
      resume *sharing* is ever added.
- [ ] **Per-user resume quota** is hardcoded at `MAX_RESUMES = 25` in
      `server/index.ts` — promote to config if needed.
- [ ] **Run `wrangler types`** (`npm run cf-typegen`) and consider replacing the
      hand-written minimal R2 type in `server/env.ts` with generated bindings.
