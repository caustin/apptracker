-- name: GetSession :one
SELECT data FROM sessions
WHERE id_hash = sqlc.arg(id_hash)
  AND (expires_at = 0 OR expires_at > sqlc.arg(now));

-- name: SetSession :exec
INSERT INTO sessions (id_hash, data, expires_at) VALUES (?, ?, ?)
ON CONFLICT (id_hash) DO UPDATE SET data = excluded.data, expires_at = excluded.expires_at;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE id_hash = ?;

-- name: DeleteExpiredSessions :execrows
DELETE FROM sessions WHERE expires_at != 0 AND expires_at <= sqlc.arg(now);
