import { useState } from 'react';
import { api } from '../api';
import type { Db, EventType, Interaction, Position, Resume, Status } from '../types';
import { EVENT_META, EVENT_TYPES, STATUSES, STATUS_META, fmtDate, fmtRange, resumeFileUrl, today } from '../types';

interface PageProps { db: Db; onChange: () => void }

const ACTIVE_ORDER: Status[] = ['offer', 'interviewing', 'screening', 'applied', 'lead', 'accepted', 'rejected', 'withdrawn'];

export function Positions({ db, onChange }: PageProps) {
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [creating, setCreating] = useState(false);

  const selected = db.positions.find((p) => p.id === selectedId);

  if (creating) {
    return (
      <PositionForm
        resumes={db.resumes}
        onCancel={() => setCreating(false)}
        onSaved={(p) => { setCreating(false); setSelectedId(p.id); onChange(); }}
      />
    );
  }

  if (selected) {
    return (
      <PositionDetail
        key={selected.id}
        position={selected}
        db={db}
        onBack={() => setSelectedId(null)}
        onChange={onChange}
      />
    );
  }

  const sorted = [...db.positions].sort(
    (a, b) => ACTIVE_ORDER.indexOf(a.status) - ACTIVE_ORDER.indexOf(b.status) || b.id - a.id,
  );

  return (
    <div>
      <div className="page-head">
        <h1>Positions</h1>
        <button className="btn btn-primary" onClick={() => setCreating(true)}>+ New position</button>
      </div>
      {sorted.length === 0 && (
        <div className="empty-state big">No positions yet. Add the first role you're pursuing.</div>
      )}
      <div className="position-list">
        {sorted.map((p) => {
          const lastEvent = db.events
            .filter((e) => e.positionId === p.id)
            .sort((a, b) => b.date.localeCompare(a.date))[0];
          return (
            <button key={p.id} className="position-row" onClick={() => setSelectedId(p.id)}>
              <div className="position-main">
                <span className="position-company">{p.company}</span>
                <span className="position-title">{p.title}</span>
              </div>
              <div className="position-meta">
                <span className="comp">{fmtRange(p.salaryMin, p.salaryMax)}</span>
                <span className="muted">{lastEvent ? `last: ${fmtDate(lastEvent.date)}` : 'no activity'}</span>
                <span className={`badge status-${p.status}`}>{STATUS_META[p.status].label}</span>
              </div>
            </button>
          );
        })}
      </div>
    </div>
  );
}

/* ---------- create / edit form ---------- */

function PositionForm({ position, resumes, onCancel, onSaved }: {
  position?: Position;
  resumes: Resume[];
  onCancel: () => void;
  onSaved: (p: Position) => void;
}) {
  const [f, setF] = useState({
    company: position?.company ?? '',
    title: position?.title ?? '',
    url: position?.url ?? '',
    location: position?.location ?? '',
    source: position?.source ?? '',
    resumeId: position?.resumeId?.toString() ?? '',
    salaryMin: position?.salaryMin?.toString() ?? '',
    salaryMax: position?.salaryMax?.toString() ?? '',
    status: position?.status ?? 'lead',
    description: position?.description ?? '',
    impressions: position?.impressions ?? '',
  });
  const set = (k: keyof typeof f) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>) =>
    setF({ ...f, [k]: e.target.value });

  const save = async () => {
    const body = {
      ...f,
      resumeId: f.resumeId ? Number(f.resumeId) : null,
      salaryMin: f.salaryMin ? Number(f.salaryMin) : null,
      salaryMax: f.salaryMax ? Number(f.salaryMax) : null,
    };
    const saved = position
      ? await api.update<Position>('positions', position.id, body)
      : await api.create<Position>('positions', body);
    onSaved(saved);
  };

  return (
    <div className="card form-card">
      <h2>{position ? 'Edit position' : 'New position'}</h2>
      <div className="form-grid">
        <label className="field">Company *<input value={f.company} onChange={set('company')} autoFocus /></label>
        <label className="field">Title *<input value={f.title} onChange={set('title')} /></label>
        <label className="field">Location<input value={f.location} onChange={set('location')} placeholder="Remote, Sydney…" /></label>
        <label className="field">Source<input value={f.source} onChange={set('source')} placeholder="LinkedIn, referral…" /></label>
        <label className="field">Posting URL<input value={f.url} onChange={set('url')} /></label>
        <label className="field">Resume used
          <select value={f.resumeId} onChange={set('resumeId')}>
            <option value="">—</option>
            {resumes.map((r) => <option key={r.id} value={r.id}>{r.name}</option>)}
          </select>
        </label>
        <label className="field">Salary min ($)<input type="number" value={f.salaryMin} onChange={set('salaryMin')} /></label>
        <label className="field">Salary max ($)<input type="number" value={f.salaryMax} onChange={set('salaryMax')} /></label>
        <label className="field">Status
          <select value={f.status} onChange={set('status')}>
            {STATUSES.map((s) => <option key={s} value={s}>{STATUS_META[s].label}</option>)}
          </select>
        </label>
      </div>
      <label className="field">Job description
        <textarea rows={6} value={f.description} onChange={set('description')} placeholder="Paste the JD here" />
      </label>
      <label className="field">Impressions
        <textarea rows={3} value={f.impressions} onChange={set('impressions')} placeholder="Gut feel, fit, red flags…" />
      </label>
      <div className="form-actions">
        <button className="btn" onClick={onCancel}>Cancel</button>
        <button className="btn btn-primary" disabled={!f.company || !f.title} onClick={save}>Save</button>
      </div>
    </div>
  );
}

/* ---------- detail ---------- */

function PositionDetail({ position, db, onBack, onChange }: {
  position: Position;
  db: Db;
  onBack: () => void;
  onChange: () => void;
}) {
  const [editing, setEditing] = useState(false);
  const posEvents = db.events
    .filter((e) => e.positionId === position.id)
    .sort((a, b) => b.date.localeCompare(a.date) || b.id - a.id);

  const setStatus = async (status: Status) => {
    await api.update('positions', position.id, { status });
    onChange();
  };

  const del = async () => {
    if (!confirm(`Delete ${position.company} — ${position.title} and all its events?`)) return;
    await api.remove('positions', position.id);
    onBack();
    onChange();
  };

  if (editing) {
    return (
      <PositionForm
        position={position}
        resumes={db.resumes}
        onCancel={() => setEditing(false)}
        onSaved={() => { setEditing(false); onChange(); }}
      />
    );
  }

  const resume = db.resumes.find((r) => r.id === position.resumeId);

  return (
    <div>
      <button className="link-btn back" onClick={onBack}>← All positions</button>
      <div className="detail-head">
        <div>
          <h1>{position.company}</h1>
          <p className="detail-title">{position.title}{position.location ? ` · ${position.location}` : ''}</p>
        </div>
        <div className="detail-actions">
          <select className="status-select" value={position.status}
            onChange={(e) => setStatus(e.target.value as Status)}>
            {STATUSES.map((s) => <option key={s} value={s}>{STATUS_META[s].label}</option>)}
          </select>
          <button className="btn" onClick={() => setEditing(true)}>Edit</button>
          <button className="btn btn-danger" onClick={del}>Delete</button>
        </div>
      </div>

      <div className="detail-facts">
        <Fact label="Compensation" value={fmtRange(position.salaryMin, position.salaryMax)} />
        <Fact label="Applied" value={fmtDate(position.appliedAt)} />
        {resume ? (
          <div className="fact">
            <div className="fact-label">Resume</div>
            <a className="fact-value" href={resumeFileUrl(resume.id)} target="_blank" rel="noreferrer">
              {resume.name} ↗
            </a>
          </div>
        ) : (
          <Fact label="Resume" value="—" />
        )}
        <Fact label="Source" value={position.source || '—'} />
        {position.url && (
          <div className="fact">
            <div className="fact-label">Posting</div>
            <a className="fact-value" href={position.url} target="_blank" rel="noreferrer">link ↗</a>
          </div>
        )}
      </div>

      <div className="detail-grid">
        <div className="detail-col">
          {position.impressions && (
            <section className="card">
              <h2>Impressions</h2>
              <p className="prose">{position.impressions}</p>
            </section>
          )}
          {position.description && (
            <section className="card">
              <h2>Job description</h2>
              <p className="prose clamp">{position.description}</p>
            </section>
          )}
        </div>

        <div className="detail-col">
          <EventForm position={position} db={db} onChange={onChange} />
          <section className="card">
            <h2>Timeline</h2>
            {posEvents.length === 0 && <div className="empty-state">No events logged yet.</div>}
            <ul className="timeline">
              {posEvents.map((e) => <EventItem key={e.id} event={e} db={db} onChange={onChange} />)}
            </ul>
          </section>
        </div>
      </div>
    </div>
  );
}

function Fact({ label, value }: { label: string; value: string }) {
  return (
    <div className="fact">
      <div className="fact-label">{label}</div>
      <div className="fact-value">{value}</div>
    </div>
  );
}

function EventItem({ event, db, onChange }: { event: Interaction; db: Db; onChange: () => void }) {
  const person = db.people.find((p) => p.id === event.personId);
  const del = async () => {
    if (!confirm('Delete this event?')) return;
    await api.remove('events', event.id);
    onChange();
  };
  return (
    <li className="timeline-item">
      <div className="timeline-head">
        <span className={`badge event-${event.type}`}>{EVENT_META[event.type]}</span>
        <span className="muted">{fmtDate(event.date)}{person ? ` · ${person.name}` : ''}</span>
        <button className="link-btn danger" onClick={del}>delete</button>
      </div>
      {event.notes && <p className="prose">{event.notes}</p>}
      {event.feedback && <p className="prose"><strong>Feedback:</strong> {event.feedback}</p>}
      {event.outcome && <p className="prose"><strong>Outcome:</strong> {event.outcome}</p>}
      {event.followupOn && <p className="followup-note">Follow up {fmtDate(event.followupOn)}</p>}
    </li>
  );
}

function EventForm({ position, db, onChange }: { position: Position; db: Db; onChange: () => void }) {
  const blank = {
    type: 'recruiter_call' as EventType,
    date: today(),
    personId: '',
    notes: '',
    feedback: '',
    outcome: '',
    followupOn: '',
  };
  const [f, setF] = useState(blank);
  const set = (k: keyof typeof f) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>) =>
    setF({ ...f, [k]: e.target.value });

  const save = async () => {
    await api.create('events', {
      ...f,
      positionId: position.id,
      personId: f.personId ? Number(f.personId) : null,
    });
    setF(blank);
    onChange();
  };

  return (
    <section className="card event-form">
      <h2>Log an event</h2>
      <div className="form-grid">
        <label className="field">Type
          <select value={f.type} onChange={set('type')}>
            {EVENT_TYPES.map((t) => <option key={t} value={t}>{EVENT_META[t]}</option>)}
          </select>
        </label>
        <label className="field">Date<input type="date" value={f.date} onChange={set('date')} /></label>
        <label className="field">With
          <select value={f.personId} onChange={set('personId')}>
            <option value="">—</option>
            {db.people.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}
          </select>
        </label>
        <label className="field">Follow up on<input type="date" value={f.followupOn} onChange={set('followupOn')} /></label>
      </div>
      <label className="field">Notes
        <textarea rows={3} value={f.notes} onChange={set('notes')} placeholder="What was discussed…" />
      </label>
      <div className="form-grid">
        <label className="field">Feedback<input value={f.feedback} onChange={set('feedback')} /></label>
        <label className="field">Outcome<input value={f.outcome} onChange={set('outcome')} placeholder="Moving forward, passed…" /></label>
      </div>
      <div className="form-actions">
        <button className="btn btn-primary" disabled={!f.date} onClick={save}>Log event</button>
      </div>
    </section>
  );
}
