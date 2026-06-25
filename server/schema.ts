import {
  pgTable,
  serial,
  text,
  integer,
  boolean,
  timestamp,
} from "drizzle-orm/pg-core";

// ---------------------------------------------------------------------------
// Better Auth tables. Field (camelCase) names must match Better Auth's schema;
// the snake_case column names are ours. Passed to the drizzle adapter in auth.ts.
// ---------------------------------------------------------------------------

export const user = pgTable("user", {
  id: text("id").primaryKey(),
  name: text("name").notNull(),
  email: text("email").notNull().unique(),
  emailVerified: boolean("email_verified").notNull().default(false),
  image: text("image"),
  createdAt: timestamp("created_at").notNull().defaultNow(),
  updatedAt: timestamp("updated_at").notNull().defaultNow(),
});

export const session = pgTable("session", {
  id: text("id").primaryKey(),
  expiresAt: timestamp("expires_at").notNull(),
  token: text("token").notNull().unique(),
  createdAt: timestamp("created_at").notNull().defaultNow(),
  updatedAt: timestamp("updated_at").notNull().defaultNow(),
  ipAddress: text("ip_address"),
  userAgent: text("user_agent"),
  userId: text("user_id")
    .notNull()
    .references(() => user.id, { onDelete: "cascade" }),
});

export const account = pgTable("account", {
  id: text("id").primaryKey(),
  accountId: text("account_id").notNull(),
  providerId: text("provider_id").notNull(),
  userId: text("user_id")
    .notNull()
    .references(() => user.id, { onDelete: "cascade" }),
  accessToken: text("access_token"),
  refreshToken: text("refresh_token"),
  idToken: text("id_token"),
  accessTokenExpiresAt: timestamp("access_token_expires_at"),
  refreshTokenExpiresAt: timestamp("refresh_token_expires_at"),
  scope: text("scope"),
  password: text("password"),
  createdAt: timestamp("created_at").notNull().defaultNow(),
  updatedAt: timestamp("updated_at").notNull().defaultNow(),
});

export const verification = pgTable("verification", {
  id: text("id").primaryKey(),
  identifier: text("identifier").notNull(),
  value: text("value").notNull(),
  expiresAt: timestamp("expires_at").notNull(),
  createdAt: timestamp("created_at").notNull().defaultNow(),
  updatedAt: timestamp("updated_at").notNull().defaultNow(),
});

// ---------------------------------------------------------------------------
// Application tables. Every row is owned by a user (per-user tenancy); dates are
// ISO YYYY-MM-DD text and money is whole-dollar integers, as before.
// ---------------------------------------------------------------------------

export const resumes = pgTable("resumes", {
  id: serial("id").primaryKey(),
  userId: text("user_id")
    .notNull()
    .references(() => user.id, { onDelete: "cascade" }),
  name: text("name").notNull(),
  // R2 object key (e.g. "<userId>/<uuid>.pdf"); never shown to the browser.
  filename: text("filename").notNull(),
  createdAt: text("created_at").notNull(),
});

export const positions = pgTable("positions", {
  id: serial("id").primaryKey(),
  userId: text("user_id")
    .notNull()
    .references(() => user.id, { onDelete: "cascade" }),
  company: text("company").notNull(),
  title: text("title").notNull(),
  url: text("url"),
  location: text("location"),
  source: text("source"),
  description: text("description"),
  impressions: text("impressions"),
  resumeId: integer("resume_id").references(() => resumes.id, {
    onDelete: "set null",
  }),
  salaryMin: integer("salary_min"),
  salaryMax: integer("salary_max"),
  status: text("status").notNull().default("lead"),
  appliedAt: text("applied_at"),
  createdAt: text("created_at").notNull(),
});

export const people = pgTable("people", {
  id: serial("id").primaryKey(),
  userId: text("user_id")
    .notNull()
    .references(() => user.id, { onDelete: "cascade" }),
  name: text("name").notNull(),
  role: text("role"),
  company: text("company"),
  email: text("email"),
  phone: text("phone"),
  linkedin: text("linkedin"),
  notes: text("notes"),
  createdAt: text("created_at").notNull(),
});

export const events = pgTable("events", {
  id: serial("id").primaryKey(),
  userId: text("user_id")
    .notNull()
    .references(() => user.id, { onDelete: "cascade" }),
  positionId: integer("position_id").references(() => positions.id, {
    onDelete: "cascade",
  }),
  personId: integer("person_id").references(() => people.id, {
    onDelete: "set null",
  }),
  type: text("type").notNull(),
  date: text("date").notNull(),
  notes: text("notes"),
  feedback: text("feedback"),
  outcome: text("outcome"),
  followupOn: text("followup_on"),
  createdAt: text("created_at").notNull(),
});

// One row per user (userId is the primary key), replacing the old id=1 singleton.
export const goals = pgTable("goals", {
  userId: text("user_id")
    .primaryKey()
    .references(() => user.id, { onDelete: "cascade" }),
  weeklyApplications: integer("weekly_applications").notNull().default(5),
  salaryMin: integer("salary_min"),
  salaryMax: integer("salary_max"),
  targetRole: text("target_role"),
  targetDate: text("target_date"),
  notes: text("notes"),
});
