-- name: GetUserByEmail :one
SELECT id, domain_id, email, password_hash, created_at
FROM users
WHERE email = $1;

-- name: GetUserById :one
SELECT id, domain_id, email, password_hash, created_at
FROM users
WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (domain_id, email, password_hash)
VALUES ($1, $2, $3)
RETURNING id, domain_id, email, password_hash, created_at;

-- name: UpdateUserImapPassword :exec
UPDATE users
SET encrypted_imap_password = $1
WHERE id = $2;
