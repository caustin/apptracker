import fs from "node:fs";
import Database from "better-sqlite3";
import { drizzle } from "drizzle-orm/better-sqlite3";

fs.mkdirSync("data/resumes", { recursive: true });

const sqlite = new Database("data/apptracker.db");
sqlite.pragma("journal_mode = WAL");
sqlite.pragma("foreign_keys = ON");

sqlite.exec(`
  CREATE TABLE IF NOT EXISTS resumes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    filename TEXT NOT NULL,
    created_at TEXT NOT NULL
  );

  CREATE TABLE IF NOT EXISTS positions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    company TEXT NOT NULL,
    title TEXT NOT NULL,
    url TEXT,
    location TEXT,
    source TEXT,
    description TEXT,
    impressions TEXT,
    resume_id INTEGER REFERENCES resumes(id) ON DELETE SET NULL,
    salary_min INTEGER,
    salary_max INTEGER,
    status TEXT NOT NULL DEFAULT 'lead',
    applied_at TEXT,
    created_at TEXT NOT NULL
  );

  CREATE TABLE IF NOT EXISTS people (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    role TEXT,
    company TEXT,
    email TEXT,
    phone TEXT,
    linkedin TEXT,
    notes TEXT,
    created_at TEXT NOT NULL
  );

  CREATE TABLE IF NOT EXISTS events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    position_id INTEGER REFERENCES positions(id) ON DELETE CASCADE,
    person_id INTEGER REFERENCES people(id) ON DELETE SET NULL,
    type TEXT NOT NULL,
    date TEXT NOT NULL,
    notes TEXT,
    feedback TEXT,
    outcome TEXT,
    followup_on TEXT,
    created_at TEXT NOT NULL
  );

  CREATE TABLE IF NOT EXISTS goals (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    weekly_applications INTEGER NOT NULL DEFAULT 5,
    salary_min INTEGER,
    salary_max INTEGER,
    target_role TEXT,
    target_date TEXT,
    notes TEXT
  );

  INSERT OR IGNORE INTO goals (id) VALUES (1);
`);

// In-place migration for databases created before the resumes table existed
// (positions used to have a free-text "resume" column instead).
const positionCols = sqlite.prepare("PRAGMA table_info(positions)").all() as {
  name: string;
}[];
if (!positionCols.some((c) => c.name === "resume_id")) {
  sqlite.exec(
    "ALTER TABLE positions ADD COLUMN resume_id INTEGER REFERENCES resumes(id) ON DELETE SET NULL",
  );
}

export const db = drizzle(sqlite);
