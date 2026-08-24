-- Controlled recipe-level milk concepts used by deterministic product matching.
-- Fat level and lactose-free status are semantic modifiers and therefore stay
-- distinct canonical ingredients. Commercial brands, package sizes and UHT
-- packaging are intentionally not part of the canonical identity.
INSERT INTO canonical_ingredients (id, name, category, subtype, default_unit) VALUES
    ('leche_entera', 'Leche entera', 'leche', 'entera', 'ml'),
    ('leche_semidesnatada', 'Leche semidesnatada', 'leche', 'semidesnatada', 'ml'),
    ('leche_desnatada', 'Leche desnatada', 'leche', 'desnatada', 'ml'),
    ('leche_entera_sin_lactosa', 'Leche entera sin lactosa', 'leche', 'entera_sin_lactosa', 'ml'),
    ('leche_semidesnatada_sin_lactosa', 'Leche semidesnatada sin lactosa', 'leche', 'semidesnatada_sin_lactosa', 'ml'),
    ('leche_desnatada_sin_lactosa', 'Leche desnatada sin lactosa', 'leche', 'desnatada_sin_lactosa', 'ml')
ON CONFLICT (id) DO NOTHING;
