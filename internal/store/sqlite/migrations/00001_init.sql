-- +goose Up
-- Dates are ISO YYYY-MM-DD text and money is whole-dollar integers, so the
-- schema ports to Postgres unchanged apart from INTEGER PRIMARY KEY.

CREATE TABLE users (
  id            TEXT PRIMARY KEY,
  email         TEXT NOT NULL UNIQUE,  -- stored lowercased
  name          TEXT NOT NULL,
  password_hash TEXT,                  -- NULL for future social-only accounts
  created_at    TEXT NOT NULL,         -- RFC 3339, UTC
  updated_at    TEXT NOT NULL
);

-- fiber.Storage for the session middleware. The key is a SHA-256 of the
-- session ID, so a leaked database does not hand out live cookies.
CREATE TABLE sessions (
  id_hash    TEXT PRIMARY KEY,
  data       BLOB NOT NULL,
  expires_at INTEGER NOT NULL          -- unix seconds; 0 = never
);
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);

CREATE TABLE resumes (
  id          INTEGER PRIMARY KEY,
  user_id     TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  name        TEXT NOT NULL,
  storage_key TEXT NOT NULL,           -- "<userId>/<uuid>.pdf"; never sent to the browser
  created_at  TEXT NOT NULL
);
CREATE INDEX resumes_user_id_idx ON resumes (user_id);

CREATE TABLE positions (
  id          INTEGER PRIMARY KEY,
  user_id     TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  company     TEXT NOT NULL,
  title       TEXT NOT NULL,
  url         TEXT,
  location    TEXT,
  source      TEXT,
  description TEXT,
  impressions TEXT,
  resume_id   INTEGER REFERENCES resumes (id) ON DELETE SET NULL,
  salary_min  INTEGER,
  salary_max  INTEGER,
  status      TEXT NOT NULL DEFAULT 'lead' CHECK (status IN (
                'lead', 'applied', 'screening', 'interviewing', 'offer',
                'accepted', 'rejected', 'withdrawn')),
  applied_at  TEXT,
  created_at  TEXT NOT NULL
);
CREATE INDEX positions_user_id_idx ON positions (user_id);
CREATE INDEX positions_resume_id_idx ON positions (resume_id);

CREATE TABLE people (
  id         INTEGER PRIMARY KEY,
  user_id    TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  name       TEXT NOT NULL,
  role       TEXT,
  company    TEXT,
  email      TEXT,
  phone      TEXT,
  linkedin   TEXT,
  notes      TEXT,
  created_at TEXT NOT NULL
);
CREATE INDEX people_user_id_idx ON people (user_id);

CREATE TABLE events (
  id          INTEGER PRIMARY KEY,
  user_id     TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  position_id INTEGER REFERENCES positions (id) ON DELETE CASCADE,
  person_id   INTEGER REFERENCES people (id) ON DELETE SET NULL,
  type        TEXT NOT NULL CHECK (type IN (
                'email', 'recruiter_call', 'interview', 'followup', 'meeting', 'note')),
  date        TEXT NOT NULL,
  notes       TEXT,
  feedback    TEXT,
  outcome     TEXT,
  followup_on TEXT,
  created_at  TEXT NOT NULL
);
CREATE INDEX events_user_id_idx ON events (user_id);
CREATE INDEX events_position_id_idx ON events (position_id);
CREATE INDEX events_person_id_idx ON events (person_id);

-- One row per user.
CREATE TABLE goals (
  user_id             TEXT PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
  weekly_applications INTEGER NOT NULL DEFAULT 5,
  salary_min          INTEGER,
  salary_max          INTEGER,
  target_role         TEXT,
  target_date         TEXT,
  notes               TEXT
);

-- +goose Down
DROP TABLE goals;
DROP TABLE events;
DROP TABLE people;
DROP TABLE positions;
DROP TABLE resumes;
DROP TABLE sessions;
DROP TABLE users;
