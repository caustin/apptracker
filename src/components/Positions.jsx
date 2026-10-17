import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { api } from "../api";
import {
  FieldError,
  FormError,
  LIMITS,
  check,
  first,
  useForm,
} from "../lib/form";
import { useAction } from "../lib/use-action";
import {
  EVENT_META,
  EVENT_TYPES,
  STATUSES,
  STATUS_META,
  fmtDate,
  fmtRange,
  resumeFileUrl,
  safeHttpHref,
  today,
} from "../constants";

const ACTIVE_ORDER = [
  "offer",
  "interviewing",
  "screening",
  "applied",
  "lead",
  "accepted",
  "rejected",
  "withdrawn",
];

export function Positions({ db, onChange }) {
  const navigate = useNavigate();
  const [creating, setCreating] = useState(false);

  if (creating) {
    return (
      <PositionForm
        resumes={db.resumes}
        onCancel={() => setCreating(false)}
        onSaved={async (p) => {
          // Load the new row before opening its page, or the page briefly
          // says "Position not found".
          await onChange();
          navigate(`/positions/${p.id}`);
        }}
      />
    );
  }

  const sorted = [...db.positions].sort(
    (a, b) =>
      ACTIVE_ORDER.indexOf(a.status) - ACTIVE_ORDER.indexOf(b.status) ||
      b.id - a.id,
  );

  return (
    <div>
      <div className="page-head">
        <h1>Positions</h1>
        <button className="btn btn-primary" onClick={() => setCreating(true)}>
          + New position
        </button>
      </div>
      {sorted.length === 0 && (
        <div className="empty-state big">
          No positions yet. Add the first role you're pursuing.
        </div>
      )}
      <div className="position-list">
        {sorted.map((p) => {
          const lastEvent = db.events
            .filter((e) => e.positionId === p.id)
            .sort((a, b) => b.date.localeCompare(a.date))[0];
          return (
            <button
              key={p.id}
              className="position-row"
              onClick={() => navigate(`/positions/${p.id}`)}
            >
              <div className="position-main">
                <span className="position-company">{p.company}</span>
                <span className="position-title">{p.title}</span>
              </div>
              <div className="position-meta">
                <span className="comp">
                  {fmtRange(p.salaryMin, p.salaryMax)}
                </span>
                <span className="muted">
                  {lastEvent
                    ? `last: ${fmtDate(lastEvent.date)}`
                    : "no activity"}
                </span>
                <span className={`badge status-${p.status}`}>
                  {STATUS_META[p.status].label}
                </span>
              </div>
            </button>
          );
        })}
      </div>
    </div>
  );
}

/* ---------- /positions/:id ---------- */

export function PositionRoute({ db, onChange }) {
  const { id } = useParams();
  const navigate = useNavigate();
  const position = db.positions.find((p) => String(p.id) === id);

  if (!position) {
    return (
      <div>
        <Link className="link-btn back" to="/positions">
          ← All positions
        </Link>
        <div className="empty-state big">Position not found.</div>
      </div>
    );
  }

  return (
    <PositionDetail
      key={position.id}
      position={position}
      db={db}
      onBack={() => navigate("/positions")}
      onChange={onChange}
    />
  );
}

/* ---------- create / edit form ---------- */

const validatePosition = (f) => ({
  company: first(
    check.required(f.company),
    check.maxLength(f.company, LIMITS.short),
  ),
  title: first(check.required(f.title), check.maxLength(f.title, LIMITS.short)),
  location: check.maxLength(f.location, LIMITS.short),
  source: check.maxLength(f.source, LIMITS.short),
  url: first(check.maxLength(f.url, LIMITS.url), check.httpUrl(f.url)),
  description: check.maxLength(f.description, LIMITS.text),
  impressions: check.maxLength(f.impressions, LIMITS.text),
  salaryMin: check.wholeDollars(f.salaryMin),
  salaryMax:
    check.wholeDollars(f.salaryMax) ||
    (f.salaryMin !== "" &&
    f.salaryMax !== "" &&
    Number(f.salaryMax) < Number(f.salaryMin)
      ? "Can't be below the minimum"
      : null),
});

function PositionForm({ position, resumes, onCancel, onSaved }) {
  const form = useForm(
    {
      company: position?.company ?? "",
      title: position?.title ?? "",
      url: position?.url ?? "",
      location: position?.location ?? "",
      source: position?.source ?? "",
      resumeId: position?.resumeId?.toString() ?? "",
      salaryMin: position?.salaryMin?.toString() ?? "",
      salaryMax: position?.salaryMax?.toString() ?? "",
      status: position?.status ?? "lead",
      description: position?.description ?? "",
      impressions: position?.impressions ?? "",
    },
    validatePosition,
  );

  const save = async (f) => {
    const body = {
      ...f,
      resumeId: f.resumeId ? Number(f.resumeId) : null,
      salaryMin: f.salaryMin ? Number(f.salaryMin) : null,
      salaryMax: f.salaryMax ? Number(f.salaryMax) : null,
    };
    const saved = position
      ? await api.update("positions", position.id, body)
      : await api.create("positions", body);
    onSaved(saved);
  };

  return (
    <form
      ref={form.ref}
      className="card form-card"
      noValidate
      onSubmit={form.submit(save)}
    >
      <h2>{position ? "Edit position" : "New position"}</h2>
      <FormError form={form} />
      <div className="form-grid">
        <label className="field">
          Company *
          <input {...form.input("company")} autoFocus />
          <FieldError form={form} name="company" />
        </label>
        <label className="field">
          Title *
          <input {...form.input("title")} />
          <FieldError form={form} name="title" />
        </label>
        <label className="field">
          Location
          <input {...form.input("location")} placeholder="Remote, Sydney…" />
          <FieldError form={form} name="location" />
        </label>
        <label className="field">
          Source
          <input {...form.input("source")} placeholder="LinkedIn, referral…" />
          <FieldError form={form} name="source" />
        </label>
        <label className="field">
          Posting URL
          <input {...form.input("url")} type="url" placeholder="https://…" />
          <FieldError form={form} name="url" />
        </label>
        <label className="field">
          Resume used
          <select {...form.input("resumeId")}>
            <option value="">—</option>
            {resumes.map((r) => (
              <option key={r.id} value={r.id}>
                {r.name}
              </option>
            ))}
          </select>
          <FieldError form={form} name="resumeId" />
        </label>
        <label className="field">
          Salary min ($)
          <input {...form.input("salaryMin")} type="number" min="0" step="1" />
          <FieldError form={form} name="salaryMin" />
        </label>
        <label className="field">
          Salary max ($)
          <input {...form.input("salaryMax")} type="number" min="0" step="1" />
          <FieldError form={form} name="salaryMax" />
        </label>
        <label className="field">
          Status
          <select {...form.input("status")}>
            {STATUSES.map((s) => (
              <option key={s} value={s}>
                {STATUS_META[s].label}
              </option>
            ))}
          </select>
        </label>
      </div>
      <label className="field">
        Job description
        <textarea
          {...form.input("description")}
          rows={6}
          placeholder="Paste the JD here"
        />
        <FieldError form={form} name="description" />
      </label>
      <label className="field">
        Impressions
        <textarea
          {...form.input("impressions")}
          rows={3}
          placeholder="Gut feel, fit, red flags…"
        />
        <FieldError form={form} name="impressions" />
      </label>
      <div className="form-actions">
        <button type="button" className="btn" onClick={onCancel}>
          Cancel
        </button>
        <button type="submit" className="btn btn-primary" disabled={form.busy}>
          {form.busy ? "Saving…" : "Save"}
        </button>
      </div>
    </form>
  );
}

/* ---------- detail ---------- */
function PositionDetail({ position, db, onBack, onChange }) {
  const [editing, setEditing] = useState(false);
  const action = useAction();
  const posEvents = db.events
    .filter((e) => e.positionId === position.id)
    .sort((a, b) => b.date.localeCompare(a.date) || b.id - a.id);
  // PUT is a full replace, so send the whole position with the new status.
  const setStatus = action.run(async (status) => {
    await api.update("positions", position.id, { ...position, status });
    onChange();
  });

  const del = action.run(async () => {
    if (
      !confirm(
        `Delete ${position.company} — ${position.title} and all its events?`,
      )
    )
      return;
    await api.remove("positions", position.id);
    onBack();
    onChange();
  });

  if (editing) {
    return (
      <PositionForm
        position={position}
        resumes={db.resumes}
        onCancel={() => setEditing(false)}
        onSaved={() => {
          setEditing(false);
          onChange();
        }}
      />
    );
  }

  const resume = db.resumes.find((r) => r.id === position.resumeId);
  return (
    <div>
      <button className="link-btn back" onClick={onBack}>
        ← All positions
      </button>
      {action.error && (
        <div className="error-banner" role="alert">
          {action.error}
        </div>
      )}
      <div className="detail-head">
        <div>
          <h1>{position.company}</h1>
          <p className="detail-title">
            {position.title}
            {position.location ? ` · ${position.location}` : ""}
          </p>
        </div>
        <div className="detail-actions">
          <select
            className="status-select"
            value={position.status}
            onChange={(e) => setStatus(e.target.value)}
          >
            {STATUSES.map((s) => (
              <option key={s} value={s}>
                {STATUS_META[s].label}
              </option>
            ))}
          </select>
          <button className="btn" onClick={() => setEditing(true)}>
            Edit
          </button>
          <button className="btn btn-danger" onClick={del}>
            Delete
          </button>
        </div>
      </div>

      <div className="detail-facts">
        <Fact
          label="Compensation"
          value={fmtRange(position.salaryMin, position.salaryMax)}
        />
        <Fact label="Applied" value={fmtDate(position.appliedAt)} />
        {resume ? (
          <div className="fact">
            <div className="fact-label">Resume</div>
            <a className="fact-value" href={resumeFileUrl(resume.id)} download>
              {resume.name} ↓
            </a>
          </div>
        ) : (
          <Fact label="Resume" value="—" />
        )}
        <Fact label="Source" value={position.source || "—"} />
        {position.url && (
          <div className="fact">
            <div className="fact-label">Posting</div>
            {safeHttpHref(position.url) ? (
              <a
                className="fact-value"
                href={safeHttpHref(position.url)}
                target="_blank"
                rel="noreferrer"
              >
                link ↗
              </a>
            ) : (
              <span className="fact-value">{position.url}</span>
            )}
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
            {posEvents.length === 0 && (
              <div className="empty-state">No events logged yet.</div>
            )}
            <ul className="timeline">
              {posEvents.map((e) => (
                <EventItem
                  key={e.id}
                  event={e}
                  db={db}
                  onChange={onChange}
                  run={action.run}
                />
              ))}
            </ul>
          </section>
        </div>
      </div>
    </div>
  );
}

function Fact({ label, value }) {
  return (
    <div className="fact">
      <div className="fact-label">{label}</div>
      <div className="fact-value">{value}</div>
    </div>
  );
}

// run reports a failed delete in the detail page's banner.
function EventItem({ event, db, onChange, run }) {
  const person = db.people.find((p) => p.id === event.personId);
  const del = run(async () => {
    if (!confirm("Delete this event?")) return;
    await api.remove("events", event.id);
    onChange();
  });

  return (
    <li className="timeline-item">
      <div className="timeline-head">
        <span className={`badge event-${event.type}`}>
          {EVENT_META[event.type]}
        </span>
        <span className="muted">
          {fmtDate(event.date)}
          {person ? ` · ${person.name}` : ""}
        </span>
        <button className="link-btn danger" onClick={del}>
          delete
        </button>
      </div>
      {event.notes && <p className="prose">{event.notes}</p>}
      {event.feedback && (
        <p className="prose">
          <strong>Feedback:</strong> {event.feedback}
        </p>
      )}
      {event.outcome && (
        <p className="prose">
          <strong>Outcome:</strong> {event.outcome}
        </p>
      )}
      {event.followupOn && (
        <p className="followup-note">Follow up {fmtDate(event.followupOn)}</p>
      )}
    </li>
  );
}

const validateEvent = (f) => ({
  type: check.required(f.type),
  notes: check.maxLength(f.notes, LIMITS.text),
  feedback: check.maxLength(f.feedback, LIMITS.short),
  outcome: check.maxLength(f.outcome, LIMITS.short),
  date: first(check.required(f.date), check.date(f.date)),
  followupOn:
    check.date(f.followupOn) ||
    (f.followupOn && f.date && f.followupOn < f.date
      ? "Can't be before the event date"
      : null),
});

function EventForm({ position, db, onChange }) {
  const blank = () => ({
    type: "recruiter_call",
    date: today(),
    personId: "",
    notes: "",
    feedback: "",
    outcome: "",
    followupOn: "",
  });

  const form = useForm(blank(), validateEvent);

  const save = async (f) => {
    await api.create("events", {
      ...f,
      positionId: position.id,
      personId: f.personId ? Number(f.personId) : null,
    });
    form.reset(blank());
    onChange();
  };

  return (
    <section className="card event-form">
      <h2>Log an event</h2>
      <form ref={form.ref} noValidate onSubmit={form.submit(save)}>
        <FormError form={form} />
        <div className="form-grid">
          <label className="field">
            Type
            <select {...form.input("type")}>
              {EVENT_TYPES.map((t) => (
                <option key={t} value={t}>
                  {EVENT_META[t]}
                </option>
              ))}
            </select>
            <FieldError form={form} name="type" />
          </label>
          <label className="field">
            Date *
            <input {...form.input("date")} type="date" />
            <FieldError form={form} name="date" />
          </label>
          <label className="field">
            With
            <select {...form.input("personId")}>
              <option value="">—</option>
              {db.people.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                </option>
              ))}
            </select>
            <FieldError form={form} name="personId" />
          </label>
          <label className="field">
            Follow up on
            <input {...form.input("followupOn")} type="date" />
            <FieldError form={form} name="followupOn" />
          </label>
        </div>
        <label className="field">
          Notes
          <textarea
            {...form.input("notes")}
            rows={3}
            placeholder="What was discussed…"
          />
          <FieldError form={form} name="notes" />
        </label>
        <div className="form-grid">
          <label className="field">
            Feedback
            <input {...form.input("feedback")} />
            <FieldError form={form} name="feedback" />
          </label>
          <label className="field">
            Outcome
            <input
              {...form.input("outcome")}
              placeholder="Moving forward, passed…"
            />
            <FieldError form={form} name="outcome" />
          </label>
        </div>
        <div className="form-actions">
          <button
            type="submit"
            className="btn btn-primary"
            disabled={form.busy}
          >
            {form.busy ? "Saving…" : "Log event"}
          </button>
        </div>
      </form>
    </section>
  );
}
