-- name: ListEvents :many
SELECT * FROM events WHERE user_id = ? ORDER BY id;

-- name: GetEvent :one
SELECT * FROM events WHERE user_id = ? AND id = ?;

-- name: CreateEvent :one
INSERT INTO events (
  user_id, position_id, person_id, type, date, notes, feedback, outcome,
  followup_on, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateEvent :one
UPDATE events
SET position_id = sqlc.narg(position_id), person_id = sqlc.narg(person_id),
    type = sqlc.arg(type), date = sqlc.arg(date), notes = sqlc.narg(notes),
    feedback = sqlc.narg(feedback), outcome = sqlc.narg(outcome),
    followup_on = sqlc.narg(followup_on)
WHERE user_id = sqlc.arg(user_id) AND id = sqlc.arg(id)
RETURNING *;

-- name: DeleteEvent :execrows
DELETE FROM events WHERE user_id = ? AND id = ?;
