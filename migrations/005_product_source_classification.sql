ALTER TABLE supermarket_products
    ADD COLUMN IF NOT EXISTS source_category_id TEXT,
    ADD COLUMN IF NOT EXISTS source_category_name TEXT,
    ADD COLUMN IF NOT EXISTS source_category_path TEXT,
    ADD COLUMN IF NOT EXISTS item_type TEXT,
    ADD COLUMN IF NOT EXISTS normalized_category TEXT,
    ADD COLUMN IF NOT EXISTS recipe_compatible BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS classification_status TEXT,
    ADD COLUMN IF NOT EXISTS classification_score NUMERIC(5,4),
    ADD COLUMN IF NOT EXISTS classification_source TEXT;

CREATE INDEX IF NOT EXISTS idx_supermarket_products_item_type
    ON supermarket_products(item_type);

CREATE INDEX IF NOT EXISTS idx_supermarket_products_normalized_category
    ON supermarket_products(normalized_category);

CREATE INDEX IF NOT EXISTS idx_supermarket_products_source_category
    ON supermarket_products(supermarket_id, source_category_id);
