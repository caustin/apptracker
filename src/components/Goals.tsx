import { useState } from "react";
import { api } from "../api";
import type { Db } from "../types";
import { fmtRange } from "../types";

export function GoalsPage({ db, onChange }: { db: Db; onChange: () => void }) {
  const g = db.goals;
  const [f, setF] = useState({
    weeklyApplications: String(g.weeklyApplications),
    salaryMin: g.salaryMin?.toString() ?? "",
    salaryMax: g.salaryMax?.toString() ?? "",
    targetRole: g.targetRole ?? "",
    targetDate: g.targetDate ?? "",
    notes: g.notes ?? "",
  });
  const [saved, setSaved] = useState(false);
  const set =
    (k: keyof typeof f) =>
    (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
      setF({ ...f, [k]: e.target.value });
      setSaved(false);
    };

  const toPositiveInt = (value: string) => {
    const n = Math.trunc(Number(value));
    return Number.isFinite(n) && n > 0 ? n : 1;
  };

  const toPositiveIntOrNull = (value: string) => {
    if (!value) return null;
    const n = Math.trunc(Number(value));
    return Number.isFinite(n) && n > 0 ? n : null;
  };

  const save = async () => {
    await api.saveGoals({
      weeklyApplications: toPositiveInt(f.weeklyApplications),
      salaryMin: toPositiveIntOrNull(f.salaryMin),
      salaryMax: toPositiveIntOrNull(f.salaryMax),
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
      <div className="card form-card">
        <h2>What does a win look like?</h2>
        <div className="form-grid">
          <label className="field">
            Target role
            <input
              value={f.targetRole}
              onChange={set("targetRole")}
              placeholder="Staff Engineer"
            />
          </label>
          <label className="field">
            Land by
            <input
              type="date"
              value={f.targetDate}
              onChange={set("targetDate")}
            />
          </label>
          <label className="field">
            Salary floor ($)
            <input
              type="number"
              step="1"
              value={f.salaryMin}
              onChange={set("salaryMin")}
            />
          </label>
          <label className="field">
            Salary target ($)
            <input
              type="number"
              step="1"
              value={f.salaryMax}
              onChange={set("salaryMax")}
            />
          </label>
          <label className="field">
            Applications per week
            <input
              type="number"
              min="1"
              step="1"
              value={f.weeklyApplications}
              onChange={set("weeklyApplications")}
            />
          </label>
        </div>
        <label className="field">
          Notes
          <textarea
            rows={3}
            value={f.notes}
            onChange={set("notes")}
            placeholder="Non-negotiables, equity expectations, must-haves…"
          />
        </label>
        <div className="form-actions">
          {saved && <span className="saved-flash">Saved ✓</span>}
          <button className="btn btn-primary" onClick={save}>
            Save goals
          </button>
        </div>
      </div>
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
