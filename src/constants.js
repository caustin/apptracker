// Shared display constants and formatters. Row shapes are defined by the
// server (see the Swagger spec at /swagger in development).

export const resumeFileUrl = (id) => `/api/v1/resumes/${id}/file`;

// Defense-in-depth for href sinks: only return the URL if it parses to an
// http(s) URL, otherwise undefined (caller renders plain text). The server
// already rejects non-http(s) schemes, but this stops any legacy/leaked row
// or future bypass from becoming a javascript:/data: XSS in the app origin.
export function safeHttpHref(u) {
  if (!u) return undefined;
  try {
    const parsed = new URL(u);
    // Return the serialized form so the value rendered to the DOM is exactly
    // what was validated (strips tolerated whitespace/control chars).
    return /^https?:$/i.test(parsed.protocol) ? parsed.href : undefined;
  } catch {
    return undefined;
  }
}

export const STATUS_META = {
  lead: { label: "Lead", order: 0 },
  applied: { label: "Applied", order: 1 },
  screening: { label: "Screening", order: 2 },
  interviewing: { label: "Interviewing", order: 3 },
  offer: { label: "Offer", order: 4 },
  accepted: { label: "Accepted", order: 5 },
  rejected: { label: "Rejected", order: -1 },
  withdrawn: { label: "Withdrawn", order: -1 },
};

export const STATUSES = Object.keys(STATUS_META);

export const EVENT_META = {
  email: "Email",
  recruiter_call: "Recruiter call",
  interview: "Interview",
  followup: "Follow-up",
  meeting: "Meeting",
  note: "Note",
};

export const EVENT_TYPES = Object.keys(EVENT_META);

// YYYY-MM-DD
export const localISO = (d) => {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
};
export const today = () => localISO(new Date());

export function fmtMoney(n) {
  if (n == null) return "—";
  return n >= 1000 ? `$${Math.round(n / 1000)}k` : `$${n}`;
}

export function fmtRange(min, max) {
  if (min == null && max == null) return "—";
  if (min != null && max != null) return `${fmtMoney(min)}–${fmtMoney(max)}`;
  return fmtMoney(min ?? max);
}

export function fmtDate(iso) {
  if (!iso) return "—";
  const [y, m, d] = iso.split("-").map(Number);
  return new Date(y, m - 1, d).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}
