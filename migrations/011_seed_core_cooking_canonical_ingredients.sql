-- High-value recipe-level concepts observed repeatedly in Gestor packs.
-- These are shopping identities, not brands, package sizes or supermarket SKUs.
-- Ambiguous recipe terms such as generic arroz, leche sin lactosa, pavo, pollo
-- and queso are intentionally NOT seeded as aliases for more specific concepts.
INSERT INTO canonical_ingredients (id, name, category, subtype, default_unit) VALUES
    ('huevo', 'Huevo', 'huevo', 'gallina', 'unidades'),
    ('pechuga_pollo', 'Pechuga de pollo', 'carne', 'pollo_pechuga', 'g'),
    ('pechuga_pavo', 'Pechuga de pavo', 'carne', 'pavo_pechuga', 'g'),
    ('aceite_oliva_virgen_extra', 'Aceite de oliva virgen extra', 'aceite', 'oliva_virgen_extra', 'ml'),
    ('yogur_natural', 'Yogur natural', 'lacteo', 'yogur_natural', 'g'),
    ('queso_fresco', 'Queso fresco', 'lacteo', 'queso_fresco', 'g'),
    ('garbanzos_cocidos', 'Garbanzos cocidos', 'legumbre', 'garbanzo_cocido', 'g'),
    ('lentejas_cocidas', 'Lentejas cocidas', 'legumbre', 'lenteja_cocida', 'g'),
    ('caldo_verduras', 'Caldo de verduras', 'caldo', 'verduras', 'ml'),
    ('caldo_pescado', 'Caldo de pescado', 'caldo', 'pescado', 'ml'),
    ('merluza', 'Merluza', 'pescado', 'merluza', 'g'),
    ('salmon', 'Salmón', 'pescado', 'salmon', 'g'),
    ('pan_integral', 'Pan integral', 'pan', 'integral', 'g'),
    ('cafe', 'Café', 'bebida', 'cafe', 'g'),
    ('aguacate', 'Aguacate', 'fruta', 'aguacate', 'g'),
    ('limon', 'Limón', 'fruta', 'limon', 'g'),
    ('platano', 'Plátano', 'fruta', 'platano', 'g')
ON CONFLICT (id) DO NOTHING;
