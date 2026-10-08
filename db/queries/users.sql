-- name: GetUserByEmail :one
SELECT *
FROM users
WHERE email = $1;

-- name: GetUserById :one
SELECT *
FROM users
WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (domain_id, email, encrypted_imap_password)
VALUES ($1, $2, $3)
RETURNING id, domain_id, email, encrypted_imap_password, auth_status, created_at;

-- name: UpdateUserImapPassword :exec
UPDATE users
SET encrypted_imap_password = $1
WHERE id = $2;

-- name: UpsertUser :one
INSERT INTO users (
    domain_id,
    email,
    encrypted_imap_password
) VALUES (
    $1, $2, $3
)
ON CONFLICT (email) DO UPDATE SET
    domain_id = EXCLUDED.domain_id,
    encrypted_imap_password = EXCLUDED.encrypted_imap_password,
    auth_status = 'active'
RETURNING id, domain_id, email, auth_status, created_at;
