-- name: CreateSession :one
INSERT INTO sessions (user_id, token_hash, ip_address, user_agent, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, user_id, token_hash, ip_address, user_agent, last_seen_at, expires_at, created_at;

-- name: GetSessionByTokenHash :one
SELECT id, user_id, token_hash, ip_address, user_agent, last_seen_at, expires_at, created_at
FROM sessions
WHERE token_hash = $1;

-- name: UpdateSessionLastSeen :one
UPDATE sessions
SET last_seen_at = CURRENT_TIMESTAMP
WHERE token_hash = $1
RETURNING id, user_id, token_hash, ip_address, user_agent, last_seen_at, expires_at, created_at;

-- name: DeleteSessionByTokenHash :exec
DELETE FROM sessions
WHERE token_hash = $1;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions
WHERE expires_at < CURRENT_TIMESTAMP;
