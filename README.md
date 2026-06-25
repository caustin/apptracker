# AppTracker

A multi-tenant job-search tracker. Each account is a private workspace: log the
positions you pursue, the recruiters and interviewers you talk to, and every
email, call, interview, and follow-up along the way. Upload the PDF resumes you
send out and attach one to each position. A dashboard shows your pipeline as a
stage funnel, weekly activity, progress against your application goals, and
upcoming follow-ups.

Runs on **Cloudflare Workers** with **Neon Postgres** (data), **R2** (resume
PDFs), and **Better Auth** (email/password + Google sign-in).

## Requirements

- Node.js 20+
- Docker (local Postgres + wsproxy, so dev needs no cloud account)

## Develop

```sh
npm install
cp .dev.vars.example .dev.vars        # edit secrets; Google is optional locally
docker compose up -d                  # local Postgres + Neon wsproxy
npm run db:migrate                    # apply migrations to the local DB
npm run dev                           # Vite (5173) + wrangler dev (8787)
```

Open http://localhost:5173. Vite proxies `/api` to the Worker, so everything is
same-origin and Better Auth cookies are first-party. Sign up with email/password
to get started. For Google sign-in locally, create an OAuth client with redirect
`http://localhost:5173/api/auth/callback/google` and fill in `.dev.vars`.

## Database changes

Schema lives in `server/schema.ts`. After editing it:

```sh
npm run db:generate                   # write a migration into drizzle/
npm run db:migrate                    # apply it
```

Mirror any table change in `src/types.ts` by hand (kept in sync deliberately).

## Deploy

One-time setup: create a Neon project, an R2 bucket named `apptracker-resumes`,
and a Google OAuth client for your production domain. Then:

```sh
wrangler secret put DATABASE_URL          # Neon connection string
wrangler secret put BETTER_AUTH_SECRET    # openssl rand -base64 32
wrangler secret put BETTER_AUTH_URL       # https://your-domain
wrangler secret put GOOGLE_CLIENT_ID
wrangler secret put GOOGLE_CLIENT_SECRET

DATABASE_URL="<neon-url>" npm run db:migrate   # migrate the prod DB
npm run deploy                                 # build + wrangler deploy
```

## Export your data

Signed in, hit **Export** (or `GET /api/export`) for a zip of all your rows
(`data.json`) plus your resume PDFs.
