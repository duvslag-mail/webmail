-- name: GetDomainByName :one
SELECT id, name, created_at
FROM DOMAINS
where name = $1;

-- name: CreateDomain :one
INSERT INTO domains (name)
VALUES ($1)
RETURNING id, name, created_at;
