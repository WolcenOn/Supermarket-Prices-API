-- Verified aliases observed in Gestor recipe packs where wording changes
-- preparation, plurality or a non-essential descriptor, but not the shopping
-- identity. We deliberately do not collapse semantic distinctions such as
-- tomate seco, tomate triturado, pimiento rojo, cebolla morada, arroz blanco,
-- generic leche sin lactosa, generic pollo/pavo or processed fish.
WITH seed(canonical_ingredient_id, alias, normalized_alias, note) AS (
    VALUES
        ('espinaca', 'Espinacas', 'espinacas', 'Plural form; same shopping ingredient.'),
        ('calabacin', 'Calabacín pelado', 'calabacin pelado', 'Peeling is recipe preparation, not a different shopping ingredient.'),
        ('zanahoria', 'Zanahoria cocida', 'zanahoria cocida', 'Cooking is recipe preparation, not a different shopping ingredient.'),
        ('judias_verdes', 'Judías verdes muy cocidas', 'judias verdes muy cocidas', 'Cooking degree is recipe preparation, not a different shopping ingredient.'),
        ('espinaca', 'Espinaca cocida', 'espinaca cocida', 'Cooking is recipe preparation, not a different shopping ingredient.'),
        ('berenjena', 'Berenjena pelada', 'berenjena pelada', 'Peeling is recipe preparation, not a different shopping ingredient.'),
        ('puerro', 'Puerro cocido', 'puerro cocido', 'Cooking is recipe preparation, not a different shopping ingredient.'),
        ('champinon', 'Champiñón muy cocido', 'champinon muy cocido', 'Cooking degree is recipe preparation, not a different shopping ingredient.'),
        ('tomate', 'Tomate maduro', 'tomate maduro', 'Ripeness descriptor does not create a separate shopping identity.'),
        ('huevo', 'Huevo cocido', 'huevo cocido', 'Cooking is recipe preparation; the shopping ingredient remains egg.'),
        ('huevo', 'Huevos', 'huevos', 'Plural form; same shopping ingredient.'),
        ('caldo_verduras', 'Caldo de verduras suave', 'caldo de verduras suave', 'Suave describes recipe intensity; shopping identity remains vegetable broth.'),
        ('caldo_pescado', 'Caldo de pescado suave', 'caldo de pescado suave', 'Suave describes recipe intensity; shopping identity remains fish broth.'),
        ('yogur_natural', 'Yogur natural sin azúcar', 'yogur natural sin azucar', 'Plain natural yoghurt canonical excludes sweetened natural yoghurt when product matching is added.'),
        ('garbanzos_cocidos', 'Garbanzo cocido', 'garbanzo cocido', 'Singular form; same cooked-legume shopping ingredient.'),
        ('lentejas_cocidas', 'Lenteja cocida', 'lenteja cocida', 'Singular form; same cooked-legume shopping ingredient.')
)
INSERT INTO canonical_ingredient_aliases (
    canonical_ingredient_id,
    alias,
    normalized_alias,
    status,
    confidence,
    decision_source,
    verification_note
)
SELECT
    canonical_ingredient_id,
    alias,
    normalized_alias,
    'verified',
    1.0,
    'pack-audit:v1',
    note
FROM seed
ON CONFLICT (canonical_ingredient_id, normalized_alias) DO UPDATE SET
    alias = EXCLUDED.alias,
    status = 'verified',
    confidence = 1.0,
    decision_source = 'pack-audit:v1',
    verification_note = EXCLUDED.verification_note,
    updated_at = NOW();

-- Keep source provenance attached to each reviewed alias.
WITH reviewed AS (
    SELECT id, alias, normalized_alias
    FROM canonical_ingredient_aliases
    WHERE decision_source = 'pack-audit:v1'
      AND status = 'verified'
)
INSERT INTO canonical_alias_evidence (
    alias_id,
    evidence_type,
    source_ref,
    source_text,
    metadata
)
SELECT
    reviewed.id,
    'pack',
    'WolcenOn/Gestor-Alimentacion:packs',
    reviewed.alias,
    jsonb_build_object('audit', 'canonical-coverage-v1', 'market', 'ES')
FROM reviewed
WHERE NOT EXISTS (
    SELECT 1
    FROM canonical_alias_evidence evidence
    WHERE evidence.alias_id = reviewed.id
      AND evidence.evidence_type = 'pack'
      AND evidence.source_ref = 'WolcenOn/Gestor-Alimentacion:packs'
);
