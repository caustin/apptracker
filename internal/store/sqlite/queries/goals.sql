-- name: GetGoals :one
SELECT * FROM goals WHERE user_id = ?;

-- name: UpsertGoals :one
INSERT INTO goals (user_id, weekly_applications, salary_min, salary_max, target_role, target_date, notes)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (user_id) DO UPDATE SET
  weekly_applications = excluded.weekly_applications,
  salary_min = excluded.salary_min,
  salary_max = excluded.salary_max,
  target_role = excluded.target_role,
  target_date = excluded.target_date,
  notes = excluded.notes
RETURNING *;
