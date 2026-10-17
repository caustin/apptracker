import { useState } from "react";
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
  fmtDate,
  safeHttpHref,
  safeMailtoHref,
} from "../constants";

const BLANK = {
  name: "",
  role: "",
  company: "",
  email: "",
  phone: "",
  linkedin: "",
  notes: "",
};

export function People({ db, onChange }) {
  const [editing, setEditing] = useState(null);
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
          key={editing === "new" ? "new" : editing.id}
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
                {p.email &&
                  (safeMailtoHref(p.email) ? (
                    <a href={safeMailtoHref(p.email)}>{p.email}</a>
                  ) : (
                    <span>{p.email}</span>
                  ))}
                {p.phone && <span>{p.phone}</span>}
                {p.linkedin &&
                  (safeHttpHref(p.linkedin) ? (
                    <a
                      href={safeHttpHref(p.linkedin)}
                      target="_blank"
                      rel="noreferrer"
                    >
                      LinkedIn ↗
                    </a>
                  ) : (
                    <span>{p.linkedin}</span>
                  ))}
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

const validatePerson = (f) => ({
  name: first(check.required(f.name), check.maxLength(f.name, LIMITS.short)),
  role: check.maxLength(f.role, LIMITS.short),
  company: check.maxLength(f.company, LIMITS.short),
  email: first(check.maxLength(f.email, LIMITS.short), check.email(f.email)),
  phone: check.maxLength(f.phone, LIMITS.short),
  linkedin: first(
    check.maxLength(f.linkedin, LIMITS.url),
    check.httpUrl(f.linkedin),
  ),
  notes: check.maxLength(f.notes, LIMITS.text),
});

function PersonForm({ person, onCancel, onSaved }) {
  const form = useForm(
    {
      name: person?.name ?? BLANK.name,
      role: person?.role ?? BLANK.role,
      company: person?.company ?? BLANK.company,
      email: person?.email ?? BLANK.email,
      phone: person?.phone ?? BLANK.phone,
      linkedin: person?.linkedin ?? BLANK.linkedin,
      notes: person?.notes ?? BLANK.notes,
    },
    validatePerson,
  );

  const save = async (f) => {
    const body = { ...f, name: f.name.trim() };
    if (person) await api.update("people", person.id, body);
    else await api.create("people", body);
    onSaved();
  };

  const action = useAction();
  const del = action.run(async () => {
    if (!person || !confirm(`Delete ${person.name}?`)) return;
    await api.remove("people", person.id);
    onSaved();
  });

  return (
    <form
      ref={form.ref}
      className="card form-card"
      noValidate
      onSubmit={form.submit(save)}
    >
      <h2>{person ? "Edit person" : "New person"}</h2>
      <FormError form={form} />
      {action.error && (
        <div className="error-banner" role="alert">
          {action.error}
        </div>
      )}
      <div className="form-grid">
        <label className="field">
          Name *
          <input {...form.input("name")} autoFocus />
          <FieldError form={form} name="name" />
        </label>
        <label className="field">
          Role
          <input
            {...form.input("role")}
            placeholder="Recruiter, hiring manager…"
          />
          <FieldError form={form} name="role" />
        </label>
        <label className="field">
          Company
          <input {...form.input("company")} />
          <FieldError form={form} name="company" />
        </label>
        <label className="field">
          Email
          <input {...form.input("email")} type="email" />
          <FieldError form={form} name="email" />
        </label>
        <label className="field">
          Phone
          <input {...form.input("phone")} type="tel" />
          <FieldError form={form} name="phone" />
        </label>
        <label className="field">
          LinkedIn
          <input
            {...form.input("linkedin")}
            type="url"
            placeholder="https://www.linkedin.com/in/…"
          />
          <FieldError form={form} name="linkedin" />
        </label>
      </div>
      <label className="field">
        Notes
        <textarea {...form.input("notes")} rows={2} />
        <FieldError form={form} name="notes" />
      </label>
      <div className="form-actions">
        {person && (
          <button type="button" className="btn btn-danger" onClick={del}>
            Delete
          </button>
        )}
        <span className="spacer" />
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
