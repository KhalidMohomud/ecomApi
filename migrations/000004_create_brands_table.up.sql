CREATE TABLE brands (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(150) NOT NULL,
    slug        VARCHAR(170) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    logo_url    VARCHAR(500),
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_brands_slug_unique ON brands (slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_brands_deleted_at ON brands (deleted_at);
