CREATE TABLE products (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          VARCHAR(200) NOT NULL,
    slug          VARCHAR(220) NOT NULL,
    sku           VARCHAR(64) NOT NULL,
    description   TEXT NOT NULL DEFAULT '',

    -- Money is stored as an integer count of the currency's smallest
    -- unit (cents for USD), never a float — floats cannot represent
    -- most decimal fractions exactly (0.1 + 0.2 != 0.3 in IEEE-754),
    -- and that kind of error compounding across a cart of items is
    -- exactly the sort of bug that's invisible in testing and painful
    -- in production. $19.99 is stored as 1999.
    price_cents   BIGINT NOT NULL CHECK (price_cents >= 0),

    -- RESTRICT is a backstop for a genuine hard DELETE, same posture
    -- as categories.parent_id's ON DELETE SET NULL (see
    -- 000003_create_categories_table.up.sql). It does NOT fire when a
    -- category is soft-deleted through the app (a soft delete is an
    -- UPDATE, not a DELETE) — that case is handled explicitly by
    -- CategoryService.Delete, which refuses to delete a category that
    -- still has products rather than silently orphaning them.
    category_id   UUID NOT NULL REFERENCES categories (id) ON DELETE RESTRICT,

    -- Unlike category_id, brand_id is optional (not every product has
    -- a distinguishable brand), so losing the reference on brand
    -- deletion is acceptable — BrandRepository.Delete nulls it out
    -- explicitly, the same "soft delete bypasses FK actions" fix
    -- applied to categories.parent_id in the previous migration.
    brand_id      UUID REFERENCES brands (id) ON DELETE SET NULL,

    is_active     BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_products_slug_unique ON products (slug) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_products_sku_unique ON products (sku) WHERE deleted_at IS NULL;
CREATE INDEX idx_products_deleted_at ON products (deleted_at);
CREATE INDEX idx_products_category_id ON products (category_id);
CREATE INDEX idx_products_brand_id ON products (brand_id);
