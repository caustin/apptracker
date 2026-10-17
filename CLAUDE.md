# AppTracker — notes for Claude

Multi-tenant job-search tracker: positions, people (contacts), events (emails, calls, interviews, follow-ups), uploaded PDF resumes, goals, and a dashboard with a stage funnel. Each user account is a private workspace — every application row carries a `user_id` and is scoped to the signed-in user.

`docs/` is **generated** by swag — never edit it by hand.

## Stack

- **Runtime:** one Go binary (`cmd/apptracker`). Fiber v3 serves the JSON API at `/api/v1/*` and the built React app, embedded from `internal/web/dist` (SPA fallback to `index.html`).
- **Frontend:** Vite + React 19 in plain JavaScript (JSX, no TypeScript), React Router v8 in declarative mode (`<BrowserRouter>`/`<Routes>`, imports from `react-router`). Plain CSS (`src/styles.css`, light mode only).
- **Database:** SQLite via `modernc.org/sqlite` (pure Go, no CGO), WAL mode, `foreign_keys=ON`, busy timeout, `_txlock=immediate` — all set in `sqlite.Open`. Migrations are goose files in `internal/store/sqlite/migrations/`, embedded and applied at startup. Queries are hand-written SQL in `internal/store/sqlite/queries/`, compiled by sqlc into `internal/store/sqlite/gen/` (never edit `gen/`).
- **Auth:** local email/password. argon2id hashes; server-side sessions via Fiber's session store, persisted in the `sessions` table (keys stored as SHA-256). No social login yet.
- **Uploads:** resume PDFs on local disk under `DATA_DIR/resumes/<userId>/<uuid>.pdf`, behind the `storage.Storage` interface; all access goes through an `os.Root`.
- **API docs:** swaggo annotations on each handler → `docs/` (Swagger 2.0). Swagger UI at `/swagger/` in development only.
- **Config:** environment only — `APP_ENV`, `APP_URL`, `DB_PATH`, `DATA_DIR`, `PORT`, `TRUST_APP_URL_ORIGIN`, `TRUSTED_PROXIES`. See `.env.example`.

## Commands

All through make (`make help`):

- `make tools` — install pinned sqlc, goose, swag, air; `npm ci`
- `make dev` — Go server with live reload (air, `:8080`) + Vite (`:5173`); Vite proxies `/api` to Go (same origin → first-party cookie, `Sec-Fetch-Site: same-origin`)
- `make run` — Go server only, serving the last frontend build
- `make build` — Vite build into `internal/web/dist`, then `go build` → `bin/apptracker`
- `make generate` (`make sqlc`, `make swagger`) — regenerate queries / API spec
- `make migrate-up|migrate-down|migrate-status` (runs `apptracker migrate …`), `make migrate-new name=…`
- `make test` — Go tests (temporary SQLite files)
- `make check` — `go vet`, gofmt, sqlc and swag output up to date, no `innerHTML` in `src/`

## Layout

- `cmd/apptracker/main.go` — config, open DB, migrate, start; `migrate` subcommand; top-level swag annotations.
- `internal/server/` — builds the Fiber app (middleware order, error handler). Used by `main` and the integration tests.
- `internal/api/` — `router.go` (routes, CSRF, rate limit, session guard, JSON 404), `dto.go` (request structs), `validate.go` (trim, empty→NULL, http(s) URL, dates, enums, IDs, JSON content type), one file per resource.
- `internal/auth/` — argon2id, sign-up/authenticate, `SignIn`/`SignOut`, `RequireUser`/`UserID`.
- `internal/domain/` — row types (JSON camelCase), status/event-type constants, `StampApplied`.
- `internal/store/` — the `Store` interface handlers depend on; `sqlite/` implements it (`store.go`), plus `sessions.go` (`fiber.Storage`) and `db.go` (open + migrations).
- `internal/storage/` — `Storage` interface + disk implementation.
- `internal/web/` — embedded frontend, security headers/CSP, Swagger mount.
- `src/` — `main.jsx` (router), `App.jsx` (session gate + routes), `api.js` (fetch wrapper, `/api/v1`, 401 handling), `constants.js` (`STATUS_META`, `EVENT_META`, formatters, `safeHttpHref`), `lib/auth-client.js` (small client keeping Better Auth's names), `lib/form.jsx` (`useForm`, `check`, `LIMITS`, `FieldError`/`FormError` for inline validation), `lib/use-action.js` (`useAction`: error banner for API calls from buttons outside a form), `components/` one file per tab.

## Conventions

- Dates are ISO `YYYY-MM-DD` strings end to end; compare them lexicographically. Server "today" is UTC. Money is whole dollars (integers, never negative).
- Position status flow: `lead → applied → screening → interviewing → offer → accepted`, with `rejected`/`withdrawn` as terminal side-exits (`order: -1` in `STATUS_META`). Enforced by Go validation and SQLite `CHECK`s.
- `appliedAt` is stamped with today the first time a position reaches `applied` or beyond; a stored value is never overwritten by the stamp, and is kept when a `PUT` omits it.
- **`PUT` is a full replace** (except that `appliedAt` rule). Clients send every field; read-only fields (`id`, `userId`, `createdAt`) in a body are ignored.
- Empty strings are stored as `NULL`. Short fields are trimmed; long text (description, notes, impressions) is kept as typed. Length caps (characters): short fields 500, URLs 2048, long text 50,000; JSON bodies over 1 MB get a 413. Contact emails must be a bare address (`net/mail`).
- Every store method takes the user ID; another user's row is indistinguishable from a missing one (404). Linked `resumeId`/`positionId`/`personId` are ownership-checked by the store in the same transaction as the write (400 `"resumeId: not found"` if not yours).
- Invalid IDs → 400; delete that matches nothing → 404; no session → 401. Errors are always `{"error": "…"}`; non-`fiber.Error`s become a bare 500 and are logged.
- Forms validate inline with `useForm` from `src/lib/form.jsx`: errors show after a field is blurred or on submit, focus moves to the first invalid field, and the server's 400 `"field: message"` errors are mapped back onto fields. Keep `check` in step with `internal/api/validate.go` when server rules change; client-only rules (e.g. max ≥ min) are fine, but never rely on the client alone.
- Routes: `/`, `/positions`, `/positions/:id`, `/people`, `/resumes`, `/goals`, `/login`. Signed-out users go to `/login`. Data flow is unchanged: `api.loadAll()` into one `db`, mutations call the API then `refresh()`.
- Resume rules: PDF only (content type or extension, plus `%PDF-` signature), 15 MB, 25 per user (count + insert in one transaction). Delete removes the file first; if that fails the row is kept and the response is 500. `positions.resume_id` is `ON DELETE SET NULL`; deleting a position cascades to its events; deleting a person clears `events.person_id`.
- Changing a table: new goose migration → update queries → `make generate` → adapt `internal/domain` and `store.go`. Never create tables at runtime.

## Security model

- **Sessions:** cookie `__Host-session` in production (`session` in development), `HttpOnly`, `SameSite=Lax`, `Secure` in prod, `Path=/`, 7-day idle / 30-day absolute timeout. Login resets the session ID (no fixation). A signed-in request re-saves the session (sliding the idle timeout) at most every 10 minutes, not on every call. Requests without a session never write a session row. `Sessions.Reset` refuses rather than deleting every user's session. Logout deletes the row server-side.
- **Passwords:** 12–128 characters, argon2id; unknown emails still run a dummy verification so timing does not reveal accounts. One generic "invalid email or password". Login + sign-up share a per-IP limiter (10/min, in-memory). Behind a reverse proxy set `TRUSTED_PROXIES` (IPs/CIDRs): `c.IP()` then walks `X-Forwarded-For` right to left past those proxies, so the limit is per client and a forged header does not help. Without it every client shares the proxy's bucket (production logs a warning).
- **CSRF (token-free):** `csrf.NewCrossOriginProtection` on all of `/api/v1`, auth included — unsafe methods pass only with `Sec-Fetch-Site: same-origin|none`, or (no header) a matching `Origin`. JSON routes require `Content-Type: application/json` (415). `GET` never changes anything. No CORS headers. **Do not add CORS, `SameSite=None`, or remove the cross-origin check without a token-based defence.** Resume upload (multipart) relies on the cross-origin check alone.
- **Headers:** every response gets `nosniff`, `Referrer-Policy`, `X-Frame-Options: DENY`, `Permissions-Policy`; HSTS in production; the strict CSP (`web.CSP`) on HTML only. Swagger gets a relaxed CSP on `/swagger/*` in development only. Adding inline script/style or a third-party host means updating `web.CSP`.
- **No `innerHTML`, ever:** `dangerouslySetInnerHTML` and `innerHTML` are banned in `src/` (`make check`). `url`/`linkedin` are http(s)-validated on the server and again on render (`safeHttpHref`).
- **Resume download:** `Content-Disposition: attachment; filename="resume.pdf"` + `nosniff` — never inline, never the user's label in a header. Storage keys are server-generated and pattern-checked; paths never come from user input.
- **Export** streams a zip (`archive/zip`), so there is no in-memory size cap.
- **Config fails closed:** in production the server refuses to start unless `APP_URL` is an `https://` origin.
- **Known limitation:** no "log out all sessions for a user" (session data is opaque). Add a version counter on `users` checked by `RequireUser` when a change-password flow arrives.
