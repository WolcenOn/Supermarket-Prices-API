CREATE TABLE canonical_ingredients (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    category TEXT,
    subtype TEXT,
    default_unit TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE ingredient_product_matches (
    canonical_ingredient_id TEXT NOT NULL REFERENCES canonical_ingredients(id) ON DELETE CASCADE,
    supermarket_product_id UUID NOT NULL REFERENCES supermarket_products(id) ON DELETE CASCADE,
    match_status TEXT NOT NULL DEFAULT 'pending'
        CHECK (match_status IN ('pending', 'automatic', 'confirmed', 'rejected')),
    match_score NUMERIC(5,4)
        CHECK (match_score IS NULL OR (match_score >= 0 AND match_score <= 1)),
    match_source TEXT NOT NULL DEFAULT 'manual',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (canonical_ingredient_id, supermarket_product_id)
);

CREATE INDEX idx_canonical_ingredients_name
    ON canonical_ingredients (LOWER(name));
CREATE INDEX idx_canonical_ingredients_category
    ON canonical_ingredients (category, subtype);
CREATE INDEX idx_ingredient_product_matches_product
    ON ingredient_product_matches (supermarket_product_id);
CREATE INDEX idx_ingredient_product_matches_status
    ON ingredient_product_matches (match_status, canonical_ingredient_id);

-- Lidl is part of the target MVP but was not included in the original seed.
INSERT INTO supermarkets (id, name) VALUES
    ('lidl', 'Lidl')
ON CONFLICT (id) DO NOTHING;

-- Small seed used to validate the canonical model with the DIA rice catalog.
-- These are recipe-level concepts, not brands or supermarket products.
INSERT INTO canonical_ingredients (id, name, category, subtype, default_unit) VALUES
    ('arroz_redondo', 'Arroz redondo', 'arroz', 'redondo', 'g'),
    ('arroz_extra', 'Arroz extra', 'arroz', 'extra', 'g'),
    ('arroz_vaporizado', 'Arroz vaporizado', 'arroz', 'vaporizado', 'g'),
    ('arroz_basmati', 'Arroz basmati', 'arroz', 'basmati', 'g'),
    ('arroz_integral', 'Arroz integral', 'arroz', 'integral', 'g')
ON CONFLICT (id) DO NOTHING;
