-- Controlled recipe-level fresh produce concepts used by deterministic DIA matching.
-- Commercial brand, package size and preparation presentation remain product-level data.
-- These canonicals intentionally stay generic where the Gestor recipe ingredient is generic;
-- more specific varieties can be added later without replacing these concepts.
INSERT INTO canonical_ingredients (id, name, category, subtype, default_unit) VALUES
    ('ajo', 'Ajo', 'verdura', 'ajo', 'g'),
    ('cebolla', 'Cebolla', 'verdura', 'cebolla', 'g'),
    ('puerro', 'Puerro', 'verdura', 'puerro', 'g'),
    ('tomate', 'Tomate', 'verdura', 'tomate', 'g'),
    ('pimiento', 'Pimiento', 'verdura', 'pimiento', 'g'),
    ('pepino', 'Pepino', 'verdura', 'pepino', 'g'),
    ('brocoli', 'Brócoli', 'verdura', 'brocoli', 'g'),
    ('coliflor', 'Coliflor', 'verdura', 'coliflor', 'g'),
    ('judias_verdes', 'Judías verdes', 'verdura', 'judias_verdes', 'g'),
    ('lechuga', 'Lechuga', 'verdura', 'lechuga', 'g'),
    ('espinaca', 'Espinaca', 'verdura', 'espinaca', 'g'),
    ('brotes_tiernos', 'Brotes tiernos', 'verdura', 'brotes_tiernos', 'g'),
    ('calabacin', 'Calabacín', 'verdura', 'calabacin', 'g'),
    ('calabaza', 'Calabaza', 'verdura', 'calabaza', 'g'),
    ('berenjena', 'Berenjena', 'verdura', 'berenjena', 'g'),
    ('patata', 'Patata', 'verdura', 'patata', 'g'),
    ('zanahoria', 'Zanahoria', 'verdura', 'zanahoria', 'g'),
    ('champinon', 'Champiñón', 'verdura', 'champinon', 'g'),
    ('seta', 'Seta', 'verdura', 'seta', 'g')
ON CONFLICT (id) DO NOTHING;
