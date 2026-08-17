CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE supermarkets (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE supermarket_products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    supermarket_id TEXT NOT NULL REFERENCES supermarkets(id),
    external_id TEXT NOT NULL,
    name TEXT NOT NULL,
    brand TEXT,
    ean TEXT,
    package_amount NUMERIC(12,3),
    package_unit TEXT,
    variable_weight BOOLEAN NOT NULL DEFAULT FALSE,
    source_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (supermarket_id, external_id)
);

CREATE TABLE price_observations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    supermarket_product_id UUID NOT NULL REFERENCES supermarket_products(id) ON DELETE CASCADE,
    postal_code TEXT,
    price NUMERIC(12,2) NOT NULL CHECK (price >= 0),
    price_per_unit NUMERIC(12,4),
    price_unit TEXT,
    currency CHAR(3) NOT NULL DEFAULT 'EUR',
    observed_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_supermarket_products_name ON supermarket_products (LOWER(name));
CREATE INDEX idx_price_observations_product_time ON price_observations (supermarket_product_id, observed_at DESC);
CREATE INDEX idx_price_observations_postal_time ON price_observations (postal_code, observed_at DESC);

INSERT INTO supermarkets (id, name) VALUES
    ('carrefour', 'Carrefour'),
    ('alcampo', 'Alcampo'),
    ('dia', 'DIA'),
    ('mercadona', 'Mercadona')
ON CONFLICT (id) DO NOTHING;
