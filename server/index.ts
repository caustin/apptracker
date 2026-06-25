import { Hono } from "hono";
import { and, eq } from "drizzle-orm";
import type { AnyPgColumn, PgTable } from "drizzle-orm/pg-core";
import { z } from "zod";
import { zipSync, strToU8 } from "fflate";
import { makeDb, type DB } from "./db";
import { makeAuth, type Auth } from "./auth";
import { r2Storage } from "./storage";
import type { Env } from "./env";
import { positions, people, events, goals, resumes } from "./schema";

type Variables = { db: DB; auth: Auth; userId: string };
const app = new Hono<{ Bindings: Env; Variables: Variables }>();

app.onError((err, c) => {
  console.error(err);
  return c.json({ error: "internal server error" }, 500);
});

const today = () => new Date().toISOString().slice(0, 10); // UTC YYYY-MM-DD
const MAX_RESUMES = 25; // per-user upload cap (keeps disk/quota bounded)
// Hard ceiling on the total resume bytes pulled into a single /api/export
// response. Workers isolates are capped at 128 MB and fflate's zipSync
// materializes the whole archive in memory, so an unbounded export can OOM
// the isolate (and impact other concurrent users on a reused isolate).
const MAX_EXPORT_BYTES = 64 * 1024 * 1024; // 64 MB

// Strict Content-Security-Policy applied to HTML responses. Vite emits
// separate JS/CSS asset files (no inline scripts/styles), so those stay
// 'self'. The only third-party host is Google Fonts (loaded via <link> in
// index.html): style-src allows the stylesheet, font-src allows the WOFF2
// files, and connect-src 'self' keeps preconnect bounded. object-src 'none'
// blocks in-origin plugin/PDF-script execution (see resume serving), and
// frame-ancestors 'none' stops clickjacking.
const CSP =
  "default-src 'self'; script-src 'self'; style-src 'self' https://fonts.googleapis.com; img-src 'self' data:; font-src 'self' https://fonts.gstatic.com; connect-src 'self'; object-src 'none'; base-uri 'self'; frame-src 'none'; frame-ancestors 'none'; form-action 'self'";

function securityHeaders(res: Response): Response {
  const headers = new Headers(res.headers);
  headers.set("X-Content-Type-Options", "nosniff");
  headers.set("Referrer-Policy", "strict-origin-when-cross-origin");
  headers.set("X-Frame-Options", "DENY");
  headers.set("Permissions-Policy", "camera=(), microphone=(), geolocation=()");
  const ct = headers.get("Content-Type") || "";
  if (ct.includes("text/html")) headers.set("Content-Security-Policy", CSP);
  return new Response(res.body, {
    status: res.status,
    statusText: res.statusText,
    headers,
  });
}

// Defense-in-depth CSRF check on mutating /api/* routes (excludes Better
// Auth's own /api/auth/* which manages its own CSRF). SameSite=Lax already
// blocks cross-site credentialed requests, but asserting the Origin/Referer
// matches the app origin guards against a future SameSite=None change. The
// Vite dev proxy keeps /api same-origin from the browser's perspective, so
// same-origin requests carry the expected Origin.
const MUTATING = new Set(["POST", "PUT", "PATCH", "DELETE"]);
function originOk(env: Env, headers: Headers): boolean {
  let expected: string;
  try {
    expected = new URL(env.BETTER_AUTH_URL).origin;
  } catch {
    // Misconfigured URL is a separate failure (makeAuth throws on this), but
    // fail closed here too so a malformed URL never widens the CSRF surface.
    return false;
  }
  const origin = headers.get("origin");
  const referer = headers.get("referer");
  let actual: string | null = origin;
  if (!actual && referer) {
    try {
      actual = new URL(referer).origin;
    } catch {
      actual = null;
    }
  }
  if (!actual) return false; // missing both → reject
  return actual === expected;
}

// Apply security headers to every response (HTML + JSON + file downloads).
app.use("*", async (c, next) => {
  await next();
  c.res = securityHeaders(c.res);
});

app.use("/api/*", async (c, next) => {
  const { db, pool } = makeDb(c.env.DATABASE_URL);
  c.set("db", db);
  try {
    // makeAuth can throw (fail-closed on a bad secret/URL); build it inside
    // the try so the finally still closes the Neon pool and doesn't leak a
    // WebSocket connection on a misconfigured deploy.
    c.set("auth", makeAuth(c.env, db));
    await next();
  } finally {
    try {
      c.executionCtx.waitUntil(pool.end());
    } catch {
      await pool.end();
    }
  }
});

// Better Auth owns /api/auth/* (sign-up/in/out, OAuth callbacks).
app.on(["GET", "POST"], "/api/auth/*", (c) => c.get("auth").handler(c.req.raw));

app.use("/api/*", async (c, next) => {
  if (c.req.path.startsWith("/api/auth/")) return next();
  // CSRF defense-in-depth on mutations (see originOk above).
  if (MUTATING.has(c.req.method) && !originOk(c.env, c.req.raw.headers))
    return c.json({ error: "origin not allowed" }, 403);
  const session = await c.get("auth").api.getSession({
    headers: c.req.raw.headers,
  });
  if (!session) return c.json({ error: "unauthorized" }, 401);
  c.set("userId", session.user.id);
  return next();
});

function parseId(c: { req: { param: (k: string) => string } }): number | null {
  const n = Number(c.req.param("id"));
  return Number.isInteger(n) && n > 0 ? n : null;
}

const zerr = (e: z.ZodError) =>
  e.issues.map((i) => `${i.path.join(".") || "body"}: ${i.message}`).join("; ");

// c.req.json() throws on malformed/empty bodies; return a sentinel so handlers
// can respond with 400 instead of letting it bubble to onError as a 500.
const BAD_JSON = Symbol("bad-json");
async function readJson(c: { req: { json: () => Promise<unknown> } }) {
  try {
    return await c.req.json();
  } catch {
    return BAD_JSON;
  }
}

// Drop keys the client omitted so DB column defaults (e.g. status='lead') apply.
function clean(obj: Record<string, unknown>) {
  for (const k of Object.keys(obj)) if (obj[k] === undefined) delete obj[k];
  return obj;
}

const safeName = (s: string) => s.replace(/[^\w.-]+/g, "_").slice(0, 80) || "resume";

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

const STATUS = [
  "lead",
  "applied",
  "screening",
  "interviewing",
  "offer",
  "accepted",
  "rejected",
  "withdrawn",
] as const;
const EVENT_TYPE = [
  "email",
  "recruiter_call",
  "interview",
  "followup",
  "meeting",
  "note",
] as const;

// Treat "" (and missing) as NULL, mirroring the old pick() helper.
const emptyToNull = <T extends z.ZodTypeAny>(schema: T) =>
  z.preprocess(
    (v) => (v === "" || v === undefined || v === null ? null : v),
    schema.nullable(),
  );
const str = () => emptyToNull(z.string().trim()).optional();
const longStr = () => emptyToNull(z.string()).optional();
const int = () => emptyToNull(z.coerce.number().int()).optional();
const refId = () => emptyToNull(z.coerce.number().int().positive()).optional();

// Free-text URL fields are rendered as <a href> on the client, so reject
// dangerous schemes (javascript:, data:, vbscript:) at the schema layer.
// Empty/missing stays NULL via emptyToNull.
const httpUrl = () =>
  emptyToNull(
    z
      .string()
      .trim()
      .url()
      .refine((s) => /^https?:$/i.test(new URL(s).protocol), {
        message: "url must be http or https",
      }),
  ).optional();

const positionCreate = z.object({
  company: z.string().trim().min(1),
  title: z.string().trim().min(1),
  url: httpUrl(),
  location: str(),
  source: str(),
  description: longStr(),
  impressions: longStr(),
  resumeId: refId(),
  salaryMin: int(),
  salaryMax: int(),
  status: z.enum(STATUS).optional(),
  appliedAt: str(),
});

const personCreate = z.object({
  name: z.string().trim().min(1),
  role: str(),
  company: str(),
  email: str(),
  phone: str(),
  linkedin: httpUrl(),
  notes: longStr(),
});

const eventCreate = z.object({
  positionId: refId(),
  personId: refId(),
  type: z.enum(EVENT_TYPE),
  date: z.string().trim().min(1),
  notes: longStr(),
  feedback: longStr(),
  outcome: longStr(),
  followupOn: str(),
});

type BeforeSave = (
  data: Record<string, unknown>,
  existing?: Record<string, unknown>,
) => void;

// Verify linked rows (resumeId, positionId, personId) belong to the current
// user, so a request can't cross-link another tenant's data. Returns an error
// string on failure, null when OK.
type VerifyRefs = (
  db: DB,
  data: Record<string, unknown>,
  uid: string,
) => Promise<string | null>;

// Stamp appliedAt the first time a position reaches "applied" or beyond.
const stampApplied: BeforeSave = (data, existing) => {
  const past = ["applied", "screening", "interviewing", "offer", "accepted"];
  if (
    typeof data.status === "string" &&
    past.includes(data.status) &&
    !data.appliedAt &&
    !existing?.appliedAt
  ) {
    data.appliedAt = today();
  }
};

const verifyPositionRefs: VerifyRefs = async (db, data, uid) => {
  if (data.resumeId != null) {
    const [r] = await db
      .select({ id: resumes.id })
      .from(resumes)
      .where(and(eq(resumes.id, data.resumeId as number), eq(resumes.userId, uid)));
    if (!r) return "resume not found";
  }
  return null;
};

const verifyEventRefs: VerifyRefs = async (db, data, uid) => {
  if (data.positionId != null) {
    const [p] = await db
      .select({ id: positions.id })
      .from(positions)
      .where(
        and(eq(positions.id, data.positionId as number), eq(positions.userId, uid)),
      );
    if (!p) return "position not found";
  }
  if (data.personId != null) {
    const [p] = await db
      .select({ id: people.id })
      .from(people)
      .where(and(eq(people.id, data.personId as number), eq(people.userId, uid)));
    if (!p) return "person not found";
  }
  return null;
};

const resources = {
  positions: {
    table: positions,
    create: positionCreate,
    update: positionCreate.partial(),
    beforeSave: stampApplied,
    verifyRefs: verifyPositionRefs,
  },
  people: { table: people, create: personCreate, update: personCreate.partial() },
  events: {
    table: events,
    create: eventCreate,
    update: eventCreate.partial(),
    verifyRefs: verifyEventRefs,
  },
} as const;

// Columns present on every owned table; lets the generic loop scope queries
// without per-operation casts on the query builder.
type Owned = { id: AnyPgColumn; userId: AnyPgColumn };

for (const [name, cfg] of Object.entries(resources)) {
  const table = cfg.table as PgTable;
  const cols = table as unknown as Owned;
  const beforeSave: BeforeSave | undefined = (
    cfg as { beforeSave?: BeforeSave }
  ).beforeSave;
  const verifyRefs: VerifyRefs | undefined = (
    cfg as { verifyRefs?: VerifyRefs }
  ).verifyRefs;

  app.get(`/api/${name}`, async (c) => {
    const rows = await c
      .get("db")
      .select()
      .from(table)
      .where(eq(cols.userId, c.get("userId")));
    return c.json(rows);
  });

  app.post(`/api/${name}`, async (c) => {
    const json = await readJson(c);
    if (json === BAD_JSON) return c.json({ error: "invalid JSON body" }, 400);
    const parsed = cfg.create.safeParse(json);
    if (!parsed.success) return c.json({ error: zerr(parsed.error) }, 400);
    const uid = c.get("userId");
    const data = clean({ ...parsed.data });
    beforeSave?.(data);
    const refErr = verifyRefs ? await verifyRefs(c.get("db"), data, uid) : null;
    if (refErr) return c.json({ error: refErr }, 400);
    data.userId = uid;
    data.createdAt = today();
    // Shape is validated above; the generic loop can't narrow the table union.
    const [row] = await c
      .get("db")
      .insert(table)
      .values(data as never)
      .returning();
    return c.json(row, 201);
  });

  app.put(`/api/${name}/:id`, async (c) => {
    const id = parseId(c);
    if (id === null) return c.json({ error: "invalid id" }, 400);
    const uid = c.get("userId");
    const db = c.get("db");
    const scope = and(eq(cols.id, id), eq(cols.userId, uid));
    const [existing] = await db.select().from(table).where(scope);
    if (!existing) return c.json({ error: "not found" }, 404);
    const json = await readJson(c);
    if (json === BAD_JSON) return c.json({ error: "invalid JSON body" }, 400);
    const parsed = cfg.update.safeParse(json);
    if (!parsed.success) return c.json({ error: zerr(parsed.error) }, 400);
    const data = clean({ ...parsed.data });
    beforeSave?.(data, existing as Record<string, unknown>);
    const refErr = verifyRefs ? await verifyRefs(db, data, uid) : null;
    if (refErr) return c.json({ error: refErr }, 400);
    if (Object.keys(data).length === 0) return c.json(existing);
    const [row] = await db
      .update(table)
      .set(data as never)
      .where(scope)
      .returning();
    return c.json(row);
  });

  app.delete(`/api/${name}/:id`, async (c) => {
    const id = parseId(c);
    if (id === null) return c.json({ error: "invalid id" }, 400);
    const deleted = await c
      .get("db")
      .delete(table)
      .where(and(eq(cols.id, id), eq(cols.userId, c.get("userId"))))
      .returning();
    if (deleted.length === 0) return c.json({ error: "not found" }, 404);
    return c.json({ ok: true });
  });
}

// ---------------------------------------------------------------------------
// Resumes (R2-backed via the Storage interface)
// ---------------------------------------------------------------------------

app.get("/api/resumes", async (c) =>
  c.json(
    await c
      .get("db")
      .select()
      .from(resumes)
      .where(eq(resumes.userId, c.get("userId"))),
  ),
);

app.post("/api/resumes", async (c) => {
  const uid = c.get("userId");
  const db = c.get("db");
  const body = await c.req.parseBody();
  const file = body.file;
  if (!(file instanceof File)) return c.json({ error: "file is required" }, 400);
  if (
    file.type !== "application/pdf" &&
    !file.name.toLowerCase().endsWith(".pdf")
  ) {
    return c.json({ error: "only PDF files are accepted" }, 400);
  }
  if (file.size > 15 * 1024 * 1024)
    return c.json({ error: "file too large (15 MB max)" }, 400);

  const owned = await db
    .select()
    .from(resumes)
    .where(eq(resumes.userId, uid));
  if (owned.length >= MAX_RESUMES)
    return c.json({ error: `resume limit reached (${MAX_RESUMES} max)` }, 400);

  const key = await r2Storage(c.env.RESUMES).put(uid, await file.arrayBuffer());
  const name =
    (typeof body.name === "string" && body.name.trim()) ||
    file.name.replace(/\.pdf$/i, "");
  try {
    const [row] = await db
      .insert(resumes)
      .values({ userId: uid, name, filename: key, createdAt: today() })
      .returning();
    return c.json(row, 201);
  } catch (err) {
    // Compensating cleanup: remove the orphaned R2 object if the DB insert
    // fails, so storage and the resumes table stay consistent.
    await r2Storage(c.env.RESUMES).delete(key).catch(() => {});
    throw err;
  }
});

app.put("/api/resumes/:id", async (c) => {
  const id = parseId(c);
  if (id === null) return c.json({ error: "invalid id" }, 400);
  const uid = c.get("userId");
  const json = await readJson(c);
  if (json === BAD_JSON) return c.json({ error: "invalid JSON body" }, 400);
  if (!json || typeof json !== "object" || Array.isArray(json))
    return c.json({ error: "request body must be a JSON object" }, 400);
  const { name } = json as { name?: string };
  if (typeof name !== "string" || !name.trim())
    return c.json({ error: "name is required" }, 400);
  const [row] = await c
    .get("db")
    .update(resumes)
    .set({ name: name.trim() })
    .where(and(eq(resumes.id, id), eq(resumes.userId, uid)))
    .returning();
  if (!row) return c.json({ error: "not found" }, 404);
  return c.json(row);
});

app.get("/api/resumes/:id/file", async (c) => {
  const id = parseId(c);
  if (id === null) return c.json({ error: "invalid id" }, 400);
  const uid = c.get("userId");
  const [row] = await c
    .get("db")
    .select()
    .from(resumes)
    .where(and(eq(resumes.id, id), eq(resumes.userId, uid)));
  if (!row) return c.json({ error: "not found" }, 404);
  const bytes = await r2Storage(c.env.RESUMES).get(row.filename);
  if (!bytes) return c.json({ error: "file missing" }, 404);
  // Static filename + nosniff: never echo the user-controlled name into headers.
  // Served as an attachment (not inline) so a malicious PDF's embedded JS
  // cannot execute in the app origin; the strict CSP also sets object-src
  // 'none' as defense in depth.
  return new Response(bytes as unknown as BodyInit, {
    headers: {
      "Content-Type": "application/pdf",
      "Content-Disposition": 'attachment; filename="resume.pdf"',
      "X-Content-Type-Options": "nosniff",
    },
  });
});

app.delete("/api/resumes/:id", async (c) => {
  const id = parseId(c);
  if (id === null) return c.json({ error: "invalid id" }, 400);
  const uid = c.get("userId");
  const db = c.get("db");
  const scope = and(eq(resumes.id, id), eq(resumes.userId, uid));
  const [row] = await db.select().from(resumes).where(scope);
  if (!row) return c.json({ error: "not found" }, 404);
  // Remove the object first; if that fails, keep the row so they stay in sync.
  try {
    await r2Storage(c.env.RESUMES).delete(row.filename);
  } catch {
    return c.json({ error: "failed to delete file" }, 500);
  }
  await db.delete(resumes).where(scope); // positions.resume_id ON DELETE SET NULL
  return c.json({ ok: true });
});

// ---------------------------------------------------------------------------
// Goals (one row per user, upserted)
// ---------------------------------------------------------------------------

const goalsDefault = (userId: string) => ({
  userId,
  weeklyApplications: 5,
  salaryMin: null,
  salaryMax: null,
  targetRole: null,
  targetDate: null,
  notes: null,
});

const goalsSchema = z.object({
  weeklyApplications: z.coerce.number().int().min(1).optional(),
  salaryMin: int(),
  salaryMax: int(),
  targetRole: str(),
  targetDate: str(),
  notes: longStr(),
});

app.get("/api/goals", async (c) => {
  const uid = c.get("userId");
  const [row] = await c
    .get("db")
    .select()
    .from(goals)
    .where(eq(goals.userId, uid));
  return c.json(row ?? goalsDefault(uid));
});

app.put("/api/goals", async (c) => {
  const uid = c.get("userId");
  const json = await readJson(c);
  if (json === BAD_JSON) return c.json({ error: "invalid JSON body" }, 400);
  const parsed = goalsSchema.safeParse(json);
  if (!parsed.success) return c.json({ error: zerr(parsed.error) }, 400);
  const data = clean({ ...parsed.data });
  const db = c.get("db");
  if (Object.keys(data).length === 0) {
    const [row] = await db.select().from(goals).where(eq(goals.userId, uid));
    return c.json(row ?? goalsDefault(uid));
  }
  const [row] = await db
    .insert(goals)
    .values({ userId: uid, ...data })
    .onConflictDoUpdate({ target: goals.userId, set: data })
    .returning();
  return c.json(row);
});

// ---------------------------------------------------------------------------
// Data export (own-your-data): all of the user's rows + their resume PDFs.
// ---------------------------------------------------------------------------

app.get("/api/export", async (c) => {
  const uid = c.get("userId");
  const db = c.get("db");
  const [pos, ppl, evs, res, gl] = await Promise.all([
    db.select().from(positions).where(eq(positions.userId, uid)),
    db.select().from(people).where(eq(people.userId, uid)),
    db.select().from(events).where(eq(events.userId, uid)),
    db.select().from(resumes).where(eq(resumes.userId, uid)),
    db
      .select()
      .from(goals)
      .where(eq(goals.userId, uid))
      .then((r) => r[0] ?? null),
  ]);
  const files: Record<string, Uint8Array> = {
    "data.json": strToU8(
      JSON.stringify(
        { positions: pos, people: ppl, events: evs, resumes: res, goals: gl },
        null,
        2,
      ),
    ),
  };
  const storage = r2Storage(c.env.RESUMES);
  // Pull resume bytes in order, but stop (413) as soon as the running total
  // would exceed the isolate-safe cap — zipSync materializes the whole
  // archive in memory, so an unbounded set can OOM the Worker.
  let total = 0;
  const fetched: { path: string; bytes: Uint8Array }[] = [];
  for (const r of res) {
    const bytes = await storage.get(r.filename);
    if (!bytes) continue;
    total += bytes.byteLength;
    if (total > MAX_EXPORT_BYTES) {
      return c.json(
        { error: "export too large; remove some resumes and try again" },
        413,
      );
    }
    fetched.push({ path: `resumes/${r.id}-${safeName(r.name)}.pdf`, bytes });
  }
  for (const f of fetched) files[f.path] = f.bytes;
  return new Response(zipSync(files) as unknown as BodyInit, {
    headers: {
      "Content-Type": "application/zip",
      "Content-Disposition": 'attachment; filename="apptracker-export.zip"',
    },
  });
});

// Unknown /api/* routes return JSON 404 instead of falling through to the
// SPA (which would leak the app shell to unauthenticated probes and return
// HTML where API clients expect JSON).
app.all("/api/*", (c) => c.json({ error: "not found" }, 404));

// Everything else: the built React app (with SPA fallback via not_found_handling).
app.all("*", (c) => c.env.ASSETS.fetch(c.req.raw));

export default app;
