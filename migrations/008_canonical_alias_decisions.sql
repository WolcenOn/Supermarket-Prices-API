CREATE TABLE canonical_alias_decisions (
    id BIGSERIAL PRIMARY KEY,
    alias_id BIGINT NOT NULL REFERENCES canonical_ingredient_aliases(id) ON DELETE CASCADE,
    from_status TEXT NOT NULL
        CHECK (from_status IN ('suggested', 'verified', 'rejected')),
    to_status TEXT NOT NULL
        CHECK (to_status IN ('verified', 'rejected')),
    decision_source TEXT NOT NULL,
    note TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_canonical_alias_decisions_alias
    ON canonical_alias_decisions (alias_id, created_at DESC);
