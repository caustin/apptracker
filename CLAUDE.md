# AppTracker — notes for Claude

Personal job-search tracker: positions, people (contacts), events (emails, calls, interviews, follow-ups), uploaded PDF resumes, goals, and a dashboard with a stage funnel.

## Stack

- **Frontend:** Vite + React 19 + TypeScript, plain CSS (`src/styles.css`, CSS variables, light mode only). No router — tab state lives in `App.tsx`.
- **Backend:** Hono on `@hono/node-server` (port 3001), Drizzle ORM over better-sqlite3 (synchronous API: `.all()` / `.get()` / `.run()`).
- **Database:** SQLite at `data/apptracker.db`. No migration tooling — tables are created with `CREATE TABLE IF NOT EXISTS` in `server/db.ts`. A schema change means editing both the DDL there and the Drizzle schema in `server/schema.ts`. For columns added after first release, follow the existing pattern in `db.ts`: check `PRAGMA table_info` and `ALTER TABLE ADD COLUMN` if missing (the user has a live database — never require deleting it).
- **Uploads:** resume PDFs live on disk in `data/resumes/` (timestamped sanitized filenames); the `resumes` table stores display name + filename. Served inline at `GET /api/resumes/:id/file`.

## Commands

- `npm run dev` — API (tsx watch) + Vite dev server with `/api` proxy, via concurrently
- `npm run build` — typecheck (`tsc --noEmit`) then `vite build`
- `npm start` — serves the API and, if `dist/` exists, the built client
- `npm run check` — typecheck only

## Layout

- `server/index.ts` — all API routes. CRUD for positions/people/events is generated from the `resources` table (field whitelist + required fields + optional `beforeSave` hook). Resumes have hand-written routes (multipart upload via `c.req.parseBody()`, PDF-only, 15 MB cap). Goals is a single row (`id = 1`), GET/PUT only.
- `server/schema.ts` — Drizzle table definitions (camelCase TS fields ↔ snake_case columns).
- `src/types.ts` — client-side copies of the row types plus shared constants (`STATUS_META`, `EVENT_META`) and formatters. **Kept in sync with `server/schema.ts` by hand** — update both when changing a table.
- `src/App.tsx` — loads everything via `api.loadAll()` into one `Db` object; mutations call the API then `refresh()`. No client-side cache or state library.
- `src/components/` — one file per tab (`Dashboard`, `Positions`, `People`, `Resumes`, `Goals`) plus `Funnel.tsx` (SVG). `Positions.tsx` also contains the detail view, position form, and event form.

## Conventions

- Dates are ISO `YYYY-MM-DD` strings end to end; compare them lexicographically. Money is whole dollars (integers).
- Position status flow: `lead → applied → screening → interviewing → offer → accepted`, with `rejected`/`withdrawn` as terminal side-exits (`order: -1` in `STATUS_META`). The funnel counts positions whose current status is at or past each stage.
- The server stamps `appliedAt` the first time a position's status moves to `applied` or beyond (see the `beforeSave` hook).
- Empty strings sent from forms are normalized to `NULL` in the API's `pick()` helper.
- The event type is named `Interaction` on the client to avoid clashing with the DOM `Event` type.
- `positions.resume_id` is `ON DELETE SET NULL` — deleting a resume unlinks it from positions rather than blocking. Deleting a resume also removes its PDF from disk.
