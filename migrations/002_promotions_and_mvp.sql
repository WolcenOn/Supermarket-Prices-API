ALTER TABLE price_observations
    ADD COLUMN IF NOT EXISTS available BOOLEAN NOT NULL DEFAULT TRUE;

CREATE TABLE IF NOT EXISTS price_promotions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    price_observation_id UUID NOT NULL REFERENCES price_observations(id) ON DELETE CASCADE,
    promotion_type TEXT NOT NULL,
    label TEXT,
    promotional_price NUMERIC(12,2),
    discount_pct NUMERIC(6,2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_price_promotions_observation
    ON price_promotions (price_observation_id);

INSERT INTO supermarkets (id, name) VALUES
    ('dia', 'DIA'),
    ('mercadona', 'Mercadona'),
    ('lidl', 'Lidl')
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name;

-- Carrefour y Alcampo no forman parte del MVP inicial. No se eliminan aquí para
-- evitar una migración destructiva en instalaciones que ya contengan datos.
