import { Pool, neonConfig } from "@neondatabase/serverless";
import { drizzle } from "drizzle-orm/neon-serverless";
import * as schema from "./schema";

// When DATABASE_URL points at a local host we're talking to a `wsproxy`
// container in front of a local Postgres (see docker-compose.yml), so the
// serverless driver must use a plain (insecure) WebSocket instead of TLS to
// Neon's cloud endpoint. Configured once; prod (neon.tech) keeps the defaults.
let localConfigured = false;
function maybeConfigureLocal(databaseUrl: string) {
  if (localConfigured) return;
  localConfigured = true;
  const host = new URL(databaseUrl).hostname;
  const isLocal =
    host === "localhost" ||
    host === "127.0.0.1" ||
    host.endsWith(".localtest.me");
  if (isLocal) {
    neonConfig.wsProxy = (h) => `${h}:4444/v1`;
    neonConfig.useSecureWebSocket = false;
    neonConfig.pipelineTLS = false;
    neonConfig.pipelineConnect = false;
  }
}

export type DB = ReturnType<typeof drizzle<typeof schema>>;

// Cloudflare bindings only exist per-request, so the client is built per-request
// from env rather than as a module singleton. Returns the pool too so the caller
// can close it once the response is sent.
export function makeDb(databaseUrl: string): { db: DB; pool: Pool } {
  maybeConfigureLocal(databaseUrl);
  const pool = new Pool({ connectionString: databaseUrl });
  const db = drizzle(pool, { schema });
  return { db, pool };
}
