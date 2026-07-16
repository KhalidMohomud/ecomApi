-- gen_random_uuid() is built into Postgres core since v13, but we
-- guard with pgcrypto for portability to older Postgres instances
-- (e.g. a self-managed VPS running an older version).
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email              VARCHAR(255) NOT NULL,
    password_hash      VARCHAR(255) NOT NULL,
    first_name         VARCHAR(100) NOT NULL,
    last_name          VARCHAR(100) NOT NULL,
    phone              VARCHAR(20),
    role               VARCHAR(20) NOT NULL DEFAULT 'customer'
                           CHECK (role IN ('customer', 'admin', 'moderator')),
    is_active          BOOLEAN NOT NULL DEFAULT true,
    email_verified_at  TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at         TIMESTAMPTZ
);

-- A partial unique index (only over rows where deleted_at IS NULL)
-- instead of a plain UNIQUE(email) constraint. This is deliberate:
-- once "Delete Account" soft-deletes a row instead of removing it,
-- a plain UNIQUE constraint would permanently block that email from
-- ever registering again, since Postgres enforces UNIQUE across all
-- rows regardless of deleted_at. A partial index enforces uniqueness
-- only among *active* accounts.
CREATE UNIQUE INDEX idx_users_email_unique ON users (email) WHERE deleted_at IS NULL;

-- Every GORM query on a soft-delete model auto-adds
-- "WHERE deleted_at IS NULL" (see internal/domain/entity.User).
-- This index keeps that filter cheap as the table grows.
CREATE INDEX idx_users_deleted_at ON users (deleted_at);
