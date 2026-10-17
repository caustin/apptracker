-- name: ListResumes :many
SELECT * FROM resumes WHERE user_id = ? ORDER BY id;

-- name: GetResume :one
SELECT * FROM resumes WHERE user_id = ? AND id = ?;

-- name: CountResumes :one
SELECT count(*) FROM resumes WHERE user_id = ?;

-- name: CreateResume :one
INSERT INTO resumes (user_id, name, storage_key, created_at)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: RenameResume :one
UPDATE resumes SET name = sqlc.arg(name)
WHERE user_id = sqlc.arg(user_id) AND id = sqlc.arg(id)
RETURNING *;

-- name: DeleteResume :execrows
DELETE FROM resumes WHERE user_id = ? AND id = ?;
