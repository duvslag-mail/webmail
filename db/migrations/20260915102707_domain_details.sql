-- +goose Up
ALTER TABLE domains
    ADD COLUMN imap_host VARCHAR(255) NOT NULL DEFAULT 'localhost',
    ADD COLUMN imap_username_format VARCHAR(20) NOT NULL DEFAULT 'full_email', -- 'full_email' or 'local_part'
    ADD COLUMN imap_port INT NOT NULL DEFAULT 993,
    ADD COLUMN imap_encryption VARCHAR(20) NOT NULL DEFAULT 'tls', -- 'tls', 'starttls', 'none'
    ADD COLUMN smtp_host VARCHAR(255) NOT NULL DEFAULT 'localhost',
    ADD COLUMN smtp_port INT NOT NULL DEFAULT 587,
    ADD COLUMN smtp_encryption VARCHAR(20) NOT NULL DEFAULT 'starttls', -- 'tls', 'starttls', 'none'
    ADD COLUMN allow_insecure_tls BOOLEAN NOT NULL DEFAULT FALSE; -- wether to allow insecure connections (self-signed certs,...)



-- +goose Down
ALTER TABLE domains
    DROP COLUMN imap_host,
    DROP COLUMN imap_port,
    DROP COLUMN imap_encryption,
    DROP COLUMN smtp_host,
    DROP COLUMN smtp_port,
    DROP COLUMN smtp_encryption,
    DROP COLUMN imap_username_format,
    DROP COLUMN allow_insecure_tls;
