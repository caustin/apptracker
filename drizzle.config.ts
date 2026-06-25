import { defineConfig } from "drizzle-kit";

// drizzle-kit connects directly over TCP (node-postgres), so locally it talks to
// the exposed Postgres on :5432 — not the wsproxy the app uses. Defaults to the
// local docker DB; set DATABASE_URL to point at Neon for prod migrations.
export default defineConfig({
  dialect: "postgresql",
  schema: "./server/schema.ts",
  out: "./drizzle",
  dbCredentials: {
    url:
      process.env.DATABASE_URL ??
      "postgres://postgres:postgres@localhost:5432/main",
  },
});
