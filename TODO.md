# AppTracker — open items

## Before production launch

- [ ] **Choose hosting** (a VM or container with a persistent disk, e.g. Fly.io
      or a VPS). This decides the Docker volume setup and whether the app sits
      on a subdomain shared with other sites (see CSRF notes in
      `planning/go-port-plan.md`).
- [ ] **Configure the proxy**: TLS, then Fiber's `TrustProxy` /
      `TrustProxyConfig` in `internal/server/server.go` so the login rate
      limiter keys on the real client IP. Set `TRUST_APP_URL_ORIGIN=true` only
      if the proxy rewrites `Host`.
- [ ] **Backups** for the SQLite file and `DATA_DIR/resumes` (e.g. Litestream
      or a nightly snapshot of the volume).
- [ ] **Verify the Docker image** builds and runs as non-root with a fresh volume.
- [ ] **Run the nightly ZAP workflow** once by hand (`workflow_dispatch`) to
      confirm the new login steps and session poll work.

## Auth

- [ ] **Google sign-in** — additive follow-up; see "Follow-up after cutover" in
      `planning/go-port-plan.md`.
- [ ] **Email verification / password reset** — needs an email provider.
- [ ] **Log out all sessions** — add a version counter on `users`, checked by
      `RequireUser`, when a change-password screen is added.

## Data

- [ ] **Old local data** — `data/apptracker.db` is from the pre-Workers app
      and is not read by the Go server (it uses `data/app.db`). Import it with
      a one-off script if you want those rows.

## Nice-to-have / later

- [ ] **Postgres store** — a second `internal/store/postgres` with its own
      migrations and sqlc config, behind the same `Store` interface.
- [ ] **Resume virus scanning** — revisit if resume sharing is ever added.
- [ ] **Per-user resume quota** is `api.MaxResumes = 25`; promote to config if needed.
