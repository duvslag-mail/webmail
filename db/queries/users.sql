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

-- name: GetUserWithDomain :one
SELECT u.email, u.encrypted_imap_password, d.imap_host, d.imap_port, d.imap_encryption, d.allow_insecure_tls
FROM users u
JOIN domains d ON u.domain_id = d.id
WHERE u.id = $1;
