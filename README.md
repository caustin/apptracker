# AppTracker

A multi-tenant job-search tracker. Each account is a private workspace: log the
positions you pursue, the recruiters and interviewers you talk to, and every
email, call, interview, and follow-up along the way. Upload the PDF resumes you
send out and attach one to each position. A dashboard shows your pipeline as a
stage funnel, weekly activity, progress against your application goals, and
upcoming follow-ups.

One Go binary (Fiber v3) serves a documented REST API at `/api/v1` and the
React app, with SQLite for data and the local disk for resume PDFs.

## Requirements

- Go 1.26+
- Node.js 22.22+ (build time only; the binary embeds the built frontend)

## Develop

```sh
make tools        # pinned sqlc, goose, swag, air + npm ci
make dev          # Go with live reload (:8080) + Vite (:5173)
```

Open http://localhost:5173 and sign up with an email and a 12+ character
password. Vite proxies `/api` to the Go server, so everything is same-origin.
The database (`data/app.db`) is created and migrated on start. API docs:
http://localhost:5173/swagger/index.html (development only).

Other targets: `make test`, `make check`, `make generate`, `make migrate-status`
— see `make help`.

## Database changes

```sh
make migrate-new name=add_things   # new goose file in internal/store/sqlite/migrations
# edit it and the queries in internal/store/sqlite/queries, then:
make generate                      # sqlc + swag
```

Migrations run automatically when the server starts.

## Build and deploy

```sh
make build                         # bin/apptracker, frontend embedded
APP_ENV=production APP_URL=https://your-domain ./bin/apptracker
```

Or with Docker (the image keeps the database and resumes on a `/data` volume):

```sh
docker build -t apptracker .
docker run -p 8080:8080 -v apptracker-data:/data \
  -e APP_URL=https://your-domain apptracker
```

Put it behind a TLS-terminating proxy; production refuses to start without an
`https://` `APP_URL`. Settings are listed in `.env.example`.

## Export your data

Signed in, hit **Export** (or `GET /api/v1/export`) for a zip of all your rows
(`data.json`) plus your resume PDFs.

## Docs

`planning/` holds the design notes and the Go port plan. `docs/` is the
generated Swagger spec.
