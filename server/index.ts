import fs from 'node:fs';
import { serve } from '@hono/node-server';
import { serveStatic } from '@hono/node-server/serve-static';
import { Hono } from 'hono';
import { eq } from 'drizzle-orm';
import type { SQLiteTable } from 'drizzle-orm/sqlite-core';
import { db } from './db';
import { positions, people, events, goals, resumes } from './schema';

const app = new Hono();
app.onError((err, c) => c.json({ error: err.message }, 500));

const today = () => new Date().toLocaleDateString('en-CA'); // local YYYY-MM-DD

function pick(body: Record<string, unknown>, fields: string[]) {
  const out: Record<string, unknown> = {};
  for (const f of fields) if (f in body) out[f] = body[f] === '' ? null : body[f];
  return out;
}

interface Resource {
  name: string;
  table: SQLiteTable;
  fields: string[];
  required: string[];
  beforeSave?: (data: Record<string, unknown>, existing?: Record<string, unknown>) => void;
}

const resources: Resource[] = [
  {
    name: 'positions',
    table: positions,
    fields: [
      'company', 'title', 'url', 'location', 'source', 'description',
      'impressions', 'resumeId', 'salaryMin', 'salaryMax', 'status', 'appliedAt',
    ],
    required: ['company', 'title'],
    beforeSave: (data, existing) => {
      // Stamp the applied date the first time a position reaches "applied" or beyond.
      const past = ['applied', 'screening', 'interviewing', 'offer', 'accepted'];
      if (
        typeof data.status === 'string' && past.includes(data.status) &&
        !data.appliedAt && !existing?.appliedAt
      ) {
        data.appliedAt = today();
      }
    },
  },
  {
    name: 'people',
    table: people,
    fields: ['name', 'role', 'company', 'email', 'phone', 'linkedin', 'notes'],
    required: ['name'],
  },
  {
    name: 'events',
    table: events,
    fields: ['positionId', 'personId', 'type', 'date', 'notes', 'feedback', 'outcome', 'followupOn'],
    required: ['type', 'date'],
  },
];

for (const { name, table, fields, required, beforeSave } of resources) {
  const t = table as never as { id: never };

  app.get(`/api/${name}`, (c) => c.json(db.select().from(table).all()));

  app.post(`/api/${name}`, async (c) => {
    const data = pick(await c.req.json(), fields);
    for (const f of required) {
      if (!data[f]) return c.json({ error: `${f} is required` }, 400);
    }
    beforeSave?.(data);
    const row = db.insert(table).values({ ...data, createdAt: today() } as never).returning().get();
    return c.json(row, 201);
  });

  app.put(`/api/${name}/:id`, async (c) => {
    const id = Number(c.req.param('id'));
    const existing = db.select().from(table).where(eq(t.id, id as never)).get();
    if (!existing) return c.json({ error: 'not found' }, 404);
    const data = pick(await c.req.json(), fields);
    beforeSave?.(data, existing as Record<string, unknown>);
    const row = db.update(table).set(data).where(eq(t.id, id as never)).returning().get();
    return c.json(row);
  });

  app.delete(`/api/${name}/:id`, (c) => {
    const id = Number(c.req.param('id'));
    db.delete(table).where(eq(t.id, id as never)).run();
    return c.json({ ok: true });
  });
}

app.get('/api/resumes', (c) => c.json(db.select().from(resumes).all()));

app.post('/api/resumes', async (c) => {
  const body = await c.req.parseBody();
  const file = body.file;
  if (!(file instanceof File)) return c.json({ error: 'file is required' }, 400);
  if (file.type !== 'application/pdf' && !file.name.toLowerCase().endsWith('.pdf')) {
    return c.json({ error: 'only PDF files are accepted' }, 400);
  }
  if (file.size > 15 * 1024 * 1024) return c.json({ error: 'file too large (15 MB max)' }, 400);

  const filename = `${Date.now()}-${file.name.replace(/[^\w.-]+/g, '_')}`;
  fs.writeFileSync(`data/resumes/${filename}`, Buffer.from(await file.arrayBuffer()));
  const name = (typeof body.name === 'string' && body.name.trim()) || file.name.replace(/\.pdf$/i, '');
  const row = db.insert(resumes).values({ name, filename, createdAt: today() }).returning().get();
  return c.json(row, 201);
});

app.put('/api/resumes/:id', async (c) => {
  const id = Number(c.req.param('id'));
  const { name } = await c.req.json();
  if (typeof name !== 'string' || !name.trim()) return c.json({ error: 'name is required' }, 400);
  const row = db.update(resumes).set({ name: name.trim() }).where(eq(resumes.id, id)).returning().get();
  if (!row) return c.json({ error: 'not found' }, 404);
  return c.json(row);
});

app.get('/api/resumes/:id/file', (c) => {
  const row = db.select().from(resumes).where(eq(resumes.id, Number(c.req.param('id')))).get();
  if (!row) return c.json({ error: 'not found' }, 404);
  const path = `data/resumes/${row.filename}`;
  if (!fs.existsSync(path)) return c.json({ error: 'file missing on disk' }, 404);
  return c.body(new Uint8Array(fs.readFileSync(path)), 200, {
    'Content-Type': 'application/pdf',
    'Content-Disposition': `inline; filename="${row.filename}"`,
  });
});

app.delete('/api/resumes/:id', (c) => {
  const id = Number(c.req.param('id'));
  const row = db.select().from(resumes).where(eq(resumes.id, id)).get();
  if (row) {
    db.delete(resumes).where(eq(resumes.id, id)).run();
    fs.rmSync(`data/resumes/${row.filename}`, { force: true });
  }
  return c.json({ ok: true });
});

app.get('/api/goals', (c) => c.json(db.select().from(goals).where(eq(goals.id, 1)).get()));

app.put('/api/goals', async (c) => {
  const data = pick(await c.req.json(), [
    'weeklyApplications', 'salaryMin', 'salaryMax', 'targetRole', 'targetDate', 'notes',
  ]);
  const row = db.update(goals).set(data).where(eq(goals.id, 1)).returning().get();
  return c.json(row);
});

// In production, serve the built client from dist/.
if (fs.existsSync('dist')) {
  app.use('/*', serveStatic({ root: './dist' }));
  app.get('*', serveStatic({ path: './dist/index.html' }));
}

const port = Number(process.env.PORT) || 3001;
serve({ fetch: app.fetch, port });
console.log(`API listening on http://localhost:${port}`);
