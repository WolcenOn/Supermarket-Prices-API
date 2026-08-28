-- Verified wording variants observed in Gestor packs for the second canonical batch.
-- Only aliases that preserve the shopping identity are included.
WITH seed(canonical_ingredient_id, alias, normalized_alias, note) AS (
    VALUES
        ('jamon_cocido', 'Jamón york', 'jamon york', 'Common Spanish recipe wording for cooked ham.'),
        ('tomate_cherry', 'Tomates cherry', 'tomates cherry', 'Plural form; same shopping ingredient.'),
        ('pimiento_rojo', 'Pimiento rojo crudo', 'pimiento rojo crudo', 'Raw describes recipe state; shopping identity remains red pepper.'),
        ('queso_parmesano', 'Parmesano', 'parmesano', 'Common shortened recipe name for Parmesan cheese.'),
        ('almendras_naturales', 'Almendras', 'almendras', 'Pack wording may omit natural when no processing is specified.'),
        ('salsa_pesto', 'Pesto', 'pesto', 'Common shortened recipe name for pesto sauce.')
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
    'pack-audit:v2',
    note
FROM seed
ON CONFLICT (canonical_ingredient_id, normalized_alias) DO UPDATE SET
    alias = EXCLUDED.alias,
    status = 'verified',
    confidence = 1.0,
    decision_source = 'pack-audit:v2',
    verification_note = EXCLUDED.verification_note,
    updated_at = NOW();

WITH reviewed AS (
    SELECT id, alias
    FROM canonical_ingredient_aliases
    WHERE decision_source = 'pack-audit:v2'
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
    jsonb_build_object('audit', 'canonical-coverage-v2', 'market', 'ES')
FROM reviewed
WHERE NOT EXISTS (
    SELECT 1
    FROM canonical_alias_evidence evidence
    WHERE evidence.alias_id = reviewed.id
      AND evidence.evidence_type = 'pack'
      AND evidence.source_ref = 'WolcenOn/Gestor-Alimentacion:packs'
);
