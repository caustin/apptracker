import type { Db, Interaction } from "../types";
import {
  EVENT_META,
  STATUS_META,
  fmtDate,
  fmtRange,
  localISO,
  today,
} from "../types";
import { Funnel } from "./Funnel";

function startOfWeek(d: Date): Date {
  const out = new Date(d);
  out.setDate(d.getDate() - ((d.getDay() + 6) % 7)); // Monday
  out.setHours(0, 0, 0, 0);
  return out;
}

const iso = localISO;

export function Dashboard({
  db,
  onNavigate,
}: {
  db: Db;
  onNavigate: (tab: "Positions" | "People" | "Goals") => void;
}) {
  const { positions, events, goals, people } = db;
  const now = today();
  const weekStart = iso(startOfWeek(new Date()));

  const active = positions.filter(
    (p) => STATUS_META[p.status].order >= 0 && p.status !== "accepted",
  );
  const appliedThisWeek = positions.filter(
    (p) => p.appliedAt && p.appliedAt >= weekStart,
  ).length;
  const interviews = events.filter((e) => e.type === "interview").length;
  const offers = positions.filter(
    (p) => p.status === "offer" || p.status === "accepted",
  ).length;

  const followups = events
    .filter((e) => e.followupOn)
    .sort((a, b) => a.followupOn!.localeCompare(b.followupOn!))
    .filter(
      (e) =>
        e.followupOn! >= now ||
        e.date >= iso(new Date(Date.now() - 30 * 86400000)),
    )
    .slice(0, 6);

  const recent = [...events]
    .sort((a, b) => b.date.localeCompare(a.date) || b.id - a.id)
    .slice(0, 6);

  const positionName = (e: Interaction) => {
    const p = positions.find((p) => p.id === e.positionId);
    return p ? `${p.company} — ${p.title}` : "General";
  };
  const personName = (e: Interaction) =>
    people.find((p) => p.id === e.personId)?.name;

  // Events per week, last 8 weeks.
  const weeks: { label: string; count: number }[] = [];
  for (let i = 7; i >= 0; i--) {
    const start = startOfWeek(new Date(Date.now() - i * 7 * 86400000));
    const end = new Date(start.getTime() + 7 * 86400000);
    weeks.push({
      label: start.toLocaleDateString("en-US", {
        month: "numeric",
        day: "numeric",
      }),
      count: events.filter((e) => e.date >= iso(start) && e.date < iso(end))
        .length,
    });
  }
  const maxWeek = Math.max(...weeks.map((w) => w.count), 1);

  const goalPct = Math.min(
    100,
    Math.round((appliedThisWeek / Math.max(goals.weeklyApplications, 1)) * 100),
  );

  return (
    <div className="dashboard">
      <div className="dash-greeting">
        <h1>Week of {fmtDate(weekStart)}</h1>
        <p className="muted">
          Target: {goals.targetRole || "any role"} ·{" "}
          {fmtRange(goals.salaryMin, goals.salaryMax)}
          {goals.targetDate && ` · land by ${fmtDate(goals.targetDate)}`}
        </p>
      </div>

      <div className="stat-row">
        <div className="stat-card">
          <div className="stat-value">{active.length}</div>
          <div className="stat-label">In pipeline</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">
            {appliedThisWeek}
            <span className="stat-of">/{goals.weeklyApplications}</span>
          </div>
          <div className="stat-label">Applied this week</div>
          <div className="goal-bar">
            <div className="goal-fill" style={{ width: `${goalPct}%` }} />
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-value">{interviews}</div>
          <div className="stat-label">Interviews logged</div>
        </div>
        <div className="stat-card accent">
          <div className="stat-value">{offers}</div>
          <div className="stat-label">Offers</div>
        </div>
      </div>

      <div className="dash-grid">
        <section className="card span2">
          <h2>The funnel</h2>
          <Funnel positions={positions} />
        </section>

        <section className="card">
          <h2>Follow-ups</h2>
          {followups.length === 0 && (
            <div className="empty-state">No follow-ups scheduled.</div>
          )}
          <ul className="list">
            {followups.map((e) => (
              <li key={e.id} className="list-item">
                <span
                  className={`date-chip ${e.followupOn! < now ? "overdue" : ""}`}
                >
                  {e.followupOn! < now ? "overdue · " : ""}
                  {fmtDate(e.followupOn)}
                </span>
                <div>
                  <div className="list-title">{positionName(e)}</div>
                  <div className="list-sub">
                    after {EVENT_META[e.type].toLowerCase()}
                    {personName(e) ? ` with ${personName(e)}` : ""}
                  </div>
                </div>
              </li>
            ))}
          </ul>
        </section>

        <section className="card span2">
          <h2>
            Activity <span className="h2-sub">events per week</span>
          </h2>
          <div className="weeks-chart">
            {weeks.map((w, i) => (
              <div key={i} className="week-col" title={`${w.count} events`}>
                <div className="week-bar-track">
                  <div
                    className="week-bar"
                    style={{ height: `${(w.count / maxWeek) * 100}%` }}
                  />
                </div>
                <div className="week-label">{w.label}</div>
              </div>
            ))}
          </div>
        </section>

        <section className="card">
          <h2>Recent activity</h2>
          {recent.length === 0 && (
            <div className="empty-state">
              No events yet.{" "}
              <button
                className="link-btn"
                onClick={() => onNavigate("Positions")}
              >
                Add a position
              </button>{" "}
              to get started.
            </div>
          )}
          <ul className="list">
            {recent.map((e) => (
              <li key={e.id} className="list-item">
                <span className="date-chip">{fmtDate(e.date)}</span>
                <div>
                  <div className="list-title">
                    {EVENT_META[e.type]} · {positionName(e)}
                  </div>
                  {e.notes && (
                    <div className="list-sub">{e.notes.slice(0, 80)}</div>
                  )}
                </div>
              </li>
            ))}
          </ul>
        </section>
      </div>
    </div>
  );
}
