export type Status =
  | "lead"
  | "applied"
  | "screening"
  | "interviewing"
  | "offer"
  | "accepted"
  | "rejected"
  | "withdrawn";

export interface Position {
  id: number;
  company: string;
  title: string;
  url: string | null;
  location: string | null;
  source: string | null;
  description: string | null;
  impressions: string | null;
  resumeId: number | null;
  salaryMin: number | null;
  salaryMax: number | null;
  status: Status;
  appliedAt: string | null;
  createdAt: string;
}

export interface Person {
  id: number;
  name: string;
  role: string | null;
  company: string | null;
  email: string | null;
  phone: string | null;
  linkedin: string | null;
  notes: string | null;
  createdAt: string;
}

export type EventType =
  | "email"
  | "recruiter_call"
  | "interview"
  | "followup"
  | "meeting"
  | "note";

export interface Interaction {
  id: number;
  positionId: number | null;
  personId: number | null;
  type: EventType;
  date: string;
  notes: string | null;
  feedback: string | null;
  outcome: string | null;
  followupOn: string | null;
  createdAt: string;
}

export interface Resume {
  id: number;
  name: string;
  filename: string;
  createdAt: string;
}

export interface Goals {
  id: number;
  weeklyApplications: number;
  salaryMin: number | null;
  salaryMax: number | null;
  targetRole: string | null;
  targetDate: string | null;
  notes: string | null;
}

export interface Db {
  positions: Position[];
  people: Person[];
  events: Interaction[];
  resumes: Resume[];
  goals: Goals;
}

export const resumeFileUrl = (id: number) => `/api/resumes/${id}/file`;

export const STATUS_META: Record<Status, { label: string; order: number }> = {
  lead: { label: "Lead", order: 0 },
  applied: { label: "Applied", order: 1 },
  screening: { label: "Screening", order: 2 },
  interviewing: { label: "Interviewing", order: 3 },
  offer: { label: "Offer", order: 4 },
  accepted: { label: "Accepted", order: 5 },
  rejected: { label: "Rejected", order: -1 },
  withdrawn: { label: "Withdrawn", order: -1 },
};

export const STATUSES = Object.keys(STATUS_META) as Status[];

export const EVENT_META: Record<EventType, string> = {
  email: "Email",
  recruiter_call: "Recruiter call",
  interview: "Interview",
  followup: "Follow-up",
  meeting: "Meeting",
  note: "Note",
};

export const EVENT_TYPES = Object.keys(EVENT_META) as EventType[];

// YYYY-MM-DD
export const localISO = (d: Date) => {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
};
export const today = () => localISO(new Date());

export function fmtMoney(n: number | null | undefined): string {
  if (n == null) return "—";
  return n >= 1000 ? `$${Math.round(n / 1000)}k` : `$${n}`;
}

export function fmtRange(min: number | null, max: number | null): string {
  if (min == null && max == null) return "—";
  if (min != null && max != null) return `${fmtMoney(min)}–${fmtMoney(max)}`;
  return fmtMoney(min ?? max);
}

export function fmtDate(iso: string | null | undefined): string {
  if (!iso) return "—";
  const [y, m, d] = iso.split("-").map(Number);
  return new Date(y, m - 1, d).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}
