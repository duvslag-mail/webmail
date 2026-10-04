-- +goose Up
SELECT 'up SQL query';

ALTER TABLE users
    DROP COLUMN password_hash,
    ADD COLUMN auth_status VARCHAR(20) NOT NULL DEFAULT 'active'; -- 'active', 'auth_failed'


-- +goose Down
SELECT 'down SQL query';

ALTER TABLE users
    ADD COLUMN password_hash VARCHAR(255) NOT NULL,
    DROP COLUMN auth_status;
