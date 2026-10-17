-- name: ListPeople :many
SELECT * FROM people WHERE user_id = ? ORDER BY id;

-- name: GetPerson :one
SELECT * FROM people WHERE user_id = ? AND id = ?;

-- name: CreatePerson :one
INSERT INTO people (user_id, name, role, company, email, phone, linkedin, notes, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdatePerson :one
UPDATE people
SET name = sqlc.arg(name), role = sqlc.narg(role), company = sqlc.narg(company),
    email = sqlc.narg(email), phone = sqlc.narg(phone), linkedin = sqlc.narg(linkedin),
    notes = sqlc.narg(notes)
WHERE user_id = sqlc.arg(user_id) AND id = sqlc.arg(id)
RETURNING *;

-- name: DeletePerson :execrows
DELETE FROM people WHERE user_id = ? AND id = ?;
