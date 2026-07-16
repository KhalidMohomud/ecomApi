CREATE TABLE refresh_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,

    -- We store a SHA-256 hash of the token, never the raw value.
    -- The raw token is returned to the client exactly once, at
    -- issuance, and never persisted anywhere. If this table ever
    -- leaked, the hashes alone cannot be used to authenticate.
    token_hash VARCHAR(64) NOT NULL,

    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_refresh_tokens_token_hash ON refresh_tokens (token_hash);

-- Supports "revoke every session for this user" (used later by
-- change-password / admin block-user flows).
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens (user_id);
