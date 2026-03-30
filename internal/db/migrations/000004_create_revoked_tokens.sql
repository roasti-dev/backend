-- +goose Up
CREATE TABLE IF NOT EXISTS revoked_tokens (
    token_hash TEXT PRIMARY KEY,
    revoked_at DATETIME NOT NULL,
    expires_at DATETIME GENERATED ALWAYS AS (datetime(revoked_at, '+90 days')) VIRTUAL
);

-- +goose Down
DROP TABLE IF EXISTS revoked_tokens;
