-- name: GetDomainByName :one
SELECT *
FROM domains
WHERE name = $1;

-- name: CreateDomain :one
INSERT INTO domains (name)
VALUES ($1)
RETURNING id, name, created_at;
