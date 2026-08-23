CREATE TABLE canonical_ingredient_aliases (
    id BIGSERIAL PRIMARY KEY,
    canonical_ingredient_id TEXT NOT NULL REFERENCES canonical_ingredients(id) ON DELETE CASCADE,
    alias TEXT NOT NULL,
    normalized_alias TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'suggested'
        CHECK (status IN ('suggested', 'verified', 'rejected')),
    confidence NUMERIC(5,4)
        CHECK (confidence IS NULL OR (confidence >= 0 AND confidence <= 1)),
    decision_source TEXT NOT NULL DEFAULT 'manual',
    verification_note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (canonical_ingredient_id, normalized_alias)
);

-- A normalized alias may be proposed for several concepts while it is under
-- review, but it may only be verified for one canonical ingredient.
CREATE UNIQUE INDEX idx_canonical_ingredient_aliases_verified
    ON canonical_ingredient_aliases (normalized_alias)
    WHERE status = 'verified';

CREATE INDEX idx_canonical_ingredient_aliases_lookup
    ON canonical_ingredient_aliases (normalized_alias, status);

CREATE TABLE canonical_alias_evidence (
    id BIGSERIAL PRIMARY KEY,
    alias_id BIGINT NOT NULL REFERENCES canonical_ingredient_aliases(id) ON DELETE CASCADE,
    evidence_type TEXT NOT NULL
        CHECK (evidence_type IN ('supermarket_product', 'source_taxonomy', 'pack', 'manual', 'rule', 'other')),
    supermarket_product_id UUID REFERENCES supermarket_products(id) ON DELETE SET NULL,
    source_ref TEXT,
    source_text TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_canonical_alias_evidence_alias
    ON canonical_alias_evidence (alias_id, created_at DESC);

CREATE INDEX idx_canonical_alias_evidence_product
    ON canonical_alias_evidence (supermarket_product_id)
    WHERE supermarket_product_id IS NOT NULL;
