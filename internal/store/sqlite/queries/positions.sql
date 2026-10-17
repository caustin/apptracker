-- name: ListPositions :many
SELECT * FROM positions WHERE user_id = ? ORDER BY id;

-- name: GetPosition :one
SELECT * FROM positions WHERE user_id = ? AND id = ?;

-- name: CreatePosition :one
INSERT INTO positions (
  user_id, company, title, url, location, source, description, impressions,
  resume_id, salary_min, salary_max, status, applied_at, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdatePosition :one
UPDATE positions
SET company = sqlc.arg(company), title = sqlc.arg(title), url = sqlc.narg(url),
    location = sqlc.narg(location), source = sqlc.narg(source),
    description = sqlc.narg(description), impressions = sqlc.narg(impressions),
    resume_id = sqlc.narg(resume_id), salary_min = sqlc.narg(salary_min),
    salary_max = sqlc.narg(salary_max), status = sqlc.arg(status),
    applied_at = sqlc.narg(applied_at)
WHERE user_id = sqlc.arg(user_id) AND id = sqlc.arg(id)
RETURNING *;

-- name: DeletePosition :execrows
DELETE FROM positions WHERE user_id = ? AND id = ?;
