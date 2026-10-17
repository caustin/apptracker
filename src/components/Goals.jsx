import { useEffect, useState } from "react";
import { api } from "../api";
import { fmtRange } from "../constants";
import {
  FieldError,
  FormError,
  LIMITS,
  check,
  first,
  useForm,
} from "../lib/form";

const validateGoals = (f) => ({
  weeklyApplications: first(
    check.required(f.weeklyApplications),
    Number.isInteger(Number(f.weeklyApplications)) &&
      Number(f.weeklyApplications) >= 1
      ? null
      : "A whole number, at least 1",
  ),
  salaryMin: check.wholeDollars(f.salaryMin),
  salaryMax:
    check.wholeDollars(f.salaryMax) ||
    (f.salaryMin !== "" &&
    f.salaryMax !== "" &&
    Number(f.salaryMax) < Number(f.salaryMin)
      ? "Can't be below the floor"
      : null),
  targetRole: check.maxLength(f.targetRole, LIMITS.short),
  targetDate: check.date(f.targetDate),
  notes: check.maxLength(f.notes, LIMITS.text),
});

export function GoalsPage({ db, onChange }) {
  const g = db.goals;
  const form = useForm(
    {
      weeklyApplications: String(g.weeklyApplications),
      salaryMin: g.salaryMin?.toString() ?? "",
      salaryMax: g.salaryMax?.toString() ?? "",
      targetRole: g.targetRole ?? "",
      targetDate: g.targetDate ?? "",
      notes: g.notes ?? "",
    },
    validateGoals,
  );
  const f = form.values;

  // "Saved ✓" goes away as soon as anything is edited.
  const [saved, setSaved] = useState(false);
  useEffect(() => setSaved(false), [f]);

  const save = async (f) => {
    await api.saveGoals({
      weeklyApplications: Number(f.weeklyApplications),
      salaryMin: f.salaryMin === "" ? null : Number(f.salaryMin),
      salaryMax: f.salaryMax === "" ? null : Number(f.salaryMax),
      targetRole: f.targetRole || null,
      targetDate: f.targetDate || null,
      notes: f.notes || null,
    });
    setSaved(true);
    onChange();
  };

  return (
    <div className="goals-page">
      <div className="page-head">
        <h1>Goals</h1>
      </div>
      <form
        ref={form.ref}
        className="card form-card"
        noValidate
        onSubmit={form.submit(save)}
      >
        <h2>What does a win look like?</h2>
        <FormError form={form} />
        <div className="form-grid">
          <label className="field">
            Target role
            <input {...form.input("targetRole")} placeholder="Staff Engineer" />
            <FieldError form={form} name="targetRole" />
          </label>
          <label className="field">
            Land by
            <input {...form.input("targetDate")} type="date" />
            <FieldError form={form} name="targetDate" />
          </label>
          <label className="field">
            Salary floor ($)
            <input
              {...form.input("salaryMin")}
              type="number"
              min="0"
              step="1"
            />
            <FieldError form={form} name="salaryMin" />
          </label>
          <label className="field">
            Salary target ($)
            <input
              {...form.input("salaryMax")}
              type="number"
              min="0"
              step="1"
            />
            <FieldError form={form} name="salaryMax" />
          </label>
          <label className="field">
            Applications per week *
            <input
              {...form.input("weeklyApplications")}
              type="number"
              min="1"
              step="1"
            />
            <FieldError form={form} name="weeklyApplications" />
          </label>
        </div>
        <label className="field">
          Notes
          <textarea
            {...form.input("notes")}
            rows={3}
            placeholder="Non-negotiables, equity expectations, must-haves…"
          />
          <FieldError form={form} name="notes" />
        </label>
        <div className="form-actions">
          {saved && <span className="saved-flash">Saved ✓</span>}
          <button
            type="submit"
            className="btn btn-primary"
            disabled={form.busy}
          >
            {form.busy ? "Saving…" : "Save goals"}
          </button>
        </div>
      </form>
      <p className="muted goal-summary">
        Aiming for {f.targetRole || "any role"} at{" "}
        {fmtRange(
          f.salaryMin ? Number(f.salaryMin) : null,
          f.salaryMax ? Number(f.salaryMax) : null,
        )}
        , {f.weeklyApplications || "?"} applications a week.
      </p>
    </div>
  );
}
