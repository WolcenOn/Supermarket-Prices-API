-- Align databases created from older versions of 001_init.sql with the
-- current catalog model. Some early installations used integer price columns,
-- which rejects valid decimal prices such as 1.20 EUR.

ALTER TABLE price_observations
    ALTER COLUMN price TYPE NUMERIC(12,2) USING price::numeric,
    ALTER COLUMN price_per_unit TYPE NUMERIC(12,4) USING price_per_unit::numeric;

ALTER TABLE supermarket_products
    ALTER COLUMN package_amount TYPE NUMERIC(12,3) USING package_amount::numeric;

ALTER TABLE price_promotions
    ALTER COLUMN promotional_price TYPE NUMERIC(12,2) USING promotional_price::numeric,
    ALTER COLUMN discount_pct TYPE NUMERIC(6,2) USING discount_pct::numeric;
