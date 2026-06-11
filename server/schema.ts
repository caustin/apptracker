import { sqliteTable, text, integer } from 'drizzle-orm/sqlite-core';

export const resumes = sqliteTable('resumes', {
  id: integer('id').primaryKey({ autoIncrement: true }),
  name: text('name').notNull(),
  filename: text('filename').notNull(),
  createdAt: text('created_at').notNull(),
});

export const positions = sqliteTable('positions', {
  id: integer('id').primaryKey({ autoIncrement: true }),
  company: text('company').notNull(),
  title: text('title').notNull(),
  url: text('url'),
  location: text('location'),
  source: text('source'),
  description: text('description'),
  impressions: text('impressions'),
  resumeId: integer('resume_id').references(() => resumes.id, { onDelete: 'set null' }),
  salaryMin: integer('salary_min'),
  salaryMax: integer('salary_max'),
  status: text('status').notNull().default('lead'),
  appliedAt: text('applied_at'),
  createdAt: text('created_at').notNull(),
});

export const people = sqliteTable('people', {
  id: integer('id').primaryKey({ autoIncrement: true }),
  name: text('name').notNull(),
  role: text('role'),
  company: text('company'),
  email: text('email'),
  phone: text('phone'),
  linkedin: text('linkedin'),
  notes: text('notes'),
  createdAt: text('created_at').notNull(),
});

export const events = sqliteTable('events', {
  id: integer('id').primaryKey({ autoIncrement: true }),
  positionId: integer('position_id').references(() => positions.id, { onDelete: 'cascade' }),
  personId: integer('person_id').references(() => people.id, { onDelete: 'set null' }),
  type: text('type').notNull(),
  date: text('date').notNull(),
  notes: text('notes'),
  feedback: text('feedback'),
  outcome: text('outcome'),
  followupOn: text('followup_on'),
  createdAt: text('created_at').notNull(),
});

export const goals = sqliteTable('goals', {
  id: integer('id').primaryKey(),
  weeklyApplications: integer('weekly_applications').notNull().default(5),
  salaryMin: integer('salary_min'),
  salaryMax: integer('salary_max'),
  targetRole: text('target_role'),
  targetDate: text('target_date'),
  notes: text('notes'),
});
