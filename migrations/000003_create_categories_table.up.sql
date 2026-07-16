CREATE TABLE categories (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(150) NOT NULL,
    slug        VARCHAR(170) NOT NULL,
    description TEXT NOT NULL DEFAULT '',

    -- Self-referencing foreign key: a category can optionally belong
    -- to a parent category (Electronics > Phones > Smartphones).
    -- ON DELETE SET NULL is a backstop for a genuine hard DELETE
    -- (there isn't one in this app today, but this keeps the
    -- constraint honest if one is ever added). The application's
    -- normal delete path is a SOFT delete, which this FK action does
    -- NOT fire for — CategoryRepository.Delete promotes children to
    -- top-level explicitly, in the same transaction, for that case.
    parent_id   UUID REFERENCES categories (id) ON DELETE SET NULL,

    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

-- Same partial-unique-index reasoning as users.email (see
-- 000001_create_users_table.up.sql): a soft-deleted category's slug
-- must be reusable by a new one.
CREATE UNIQUE INDEX idx_categories_slug_unique ON categories (slug) WHERE deleted_at IS NULL;

CREATE INDEX idx_categories_deleted_at ON categories (deleted_at);
CREATE INDEX idx_categories_parent_id ON categories (parent_id);
