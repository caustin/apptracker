import { useState } from "react";
import { api } from "../api";
import type { Db, Person } from "../types";
import { EVENT_META, fmtDate } from "../types";

const BLANK = {
  name: "",
  role: "",
  company: "",
  email: "",
  phone: "",
  linkedin: "",
  notes: "",
};

export function People({ db, onChange }: { db: Db; onChange: () => void }) {
  const [editing, setEditing] = useState<Person | "new" | null>(null);

  return (
    <div>
      <div className="page-head">
        <h1>People</h1>
        <button className="btn btn-primary" onClick={() => setEditing("new")}>
          + New person
        </button>
      </div>

      {editing && (
        <PersonForm
          person={editing === "new" ? undefined : editing}
          onCancel={() => setEditing(null)}
          onSaved={() => {
            setEditing(null);
            onChange();
          }}
        />
      )}

      {db.people.length === 0 && !editing && (
        <div className="empty-state big">
          No contacts yet. Add the recruiters and interviewers you talk to.
        </div>
      )}

      <div className="people-grid">
        {db.people.map((p) => {
          const lastEvent = db.events
            .filter((e) => e.personId === p.id)
            .sort((a, b) => b.date.localeCompare(a.date))[0];
          return (
            <div key={p.id} className="card person-card">
              <div className="person-head">
                <span className="person-name">{p.name}</span>
                <button className="link-btn" onClick={() => setEditing(p)}>
                  edit
                </button>
              </div>
              <div className="person-role">
                {[p.role, p.company].filter(Boolean).join(" · ") || "—"}
              </div>
              <div className="person-contact">
                {p.email && <a href={`mailto:${p.email}`}>{p.email}</a>}
                {p.phone && <span>{p.phone}</span>}
                {p.linkedin && (
                  <a href={p.linkedin} target="_blank" rel="noreferrer">
                    LinkedIn ↗
                  </a>
                )}
              </div>
              {p.notes && <p className="prose">{p.notes}</p>}
              {lastEvent && (
                <div className="muted small">
                  Last contact: {EVENT_META[lastEvent.type].toLowerCase()},{" "}
                  {fmtDate(lastEvent.date)}
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}

function PersonForm({
  person,
  onCancel,
  onSaved,
}: {
  person?: Person;
  onCancel: () => void;
  onSaved: () => void;
}) {
  const [f, setF] = useState({
    name: person?.name ?? BLANK.name,
    role: person?.role ?? BLANK.role,
    company: person?.company ?? BLANK.company,
    email: person?.email ?? BLANK.email,
    phone: person?.phone ?? BLANK.phone,
    linkedin: person?.linkedin ?? BLANK.linkedin,
    notes: person?.notes ?? BLANK.notes,
  });
  const set =
    (k: keyof typeof f) =>
    (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
      setF({ ...f, [k]: e.target.value });

  const save = async () => {
    const body = { ...f, name: f.name.trim() };
    if (person) await api.update("people", person.id, body);
    else await api.create("people", body);
    onSaved();
  };

  const del = async () => {
    if (!person || !confirm(`Delete ${person.name}?`)) return;
    await api.remove("people", person.id);
    onSaved();
  };

  return (
    <div className="card form-card">
      <h2>{person ? "Edit person" : "New person"}</h2>
      <div className="form-grid">
        <label className="field">
          Name *<input value={f.name} onChange={set("name")} autoFocus />
        </label>
        <label className="field">
          Role
          <input
            value={f.role}
            onChange={set("role")}
            placeholder="Recruiter, hiring manager…"
          />
        </label>
        <label className="field">
          Company
          <input value={f.company} onChange={set("company")} />
        </label>
        <label className="field">
          Email
          <input value={f.email} onChange={set("email")} />
        </label>
        <label className="field">
          Phone
          <input value={f.phone} onChange={set("phone")} />
        </label>
        <label className="field">
          LinkedIn
          <input value={f.linkedin} onChange={set("linkedin")} />
        </label>
      </div>
      <label className="field">
        Notes
        <textarea rows={2} value={f.notes} onChange={set("notes")} />
      </label>
      <div className="form-actions">
        {person && (
          <button className="btn btn-danger" onClick={del}>
            Delete
          </button>
        )}
        <span className="spacer" />
        <button className="btn" onClick={onCancel}>
          Cancel
        </button>
        <button
          className="btn btn-primary"
          disabled={!f.name.trim()}
          onClick={save}
        >
          Save
        </button>
      </div>
    </div>
  );
}
