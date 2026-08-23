package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	canonicaltext "github.com/WolcenOn/Supermarket-Prices-API/internal/canonical"
	"github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

type CanonicalAliasProposalInput struct {
	CanonicalIngredientID string
	Alias                 string
	Confidence            float64
	DecisionSource        string
	EvidenceType          string
	EvidenceSourceRef     string
	EvidenceSourceText    string
}

type CanonicalAliasProposalPreview struct {
	CanonicalIngredient catalog.CanonicalIngredient  `json:"canonicalIngredient"`
	Alias               string                       `json:"alias"`
	NormalizedAlias     string                       `json:"normalizedAlias"`
	AlreadyCanonical    bool                         `json:"alreadyCanonical"`
	ExistingStatus      string                       `json:"existingStatus,omitempty"`
	VerifiedConflict    *catalog.CanonicalIngredient `json:"verifiedConflict,omitempty"`
}

type SavedCanonicalAliasProposal struct {
	AliasID int64  `json:"aliasId"`
	Status  string `json:"status"`
}

func PreviewCanonicalAliasProposal(ctx context.Context, db *sql.DB, canonicalIngredientID, alias string) (CanonicalAliasProposalPreview, error) {
	if db == nil {
		return CanonicalAliasProposalPreview{}, fmt.Errorf("postgres canonical alias proposal: database is required")
	}

	canonicalIngredientID = strings.TrimSpace(canonicalIngredientID)
	alias = strings.TrimSpace(alias)
	normalizedAlias := canonicaltext.NormalizeText(alias)
	if canonicalIngredientID == "" || normalizedAlias == "" {
		return CanonicalAliasProposalPreview{}, fmt.Errorf("postgres canonical alias proposal: canonical ingredient id and alias are required")
	}

	preview := CanonicalAliasProposalPreview{
		Alias:           alias,
		NormalizedAlias: normalizedAlias,
	}

	if err := db.QueryRowContext(ctx, `
		SELECT id, name, COALESCE(category, ''), COALESCE(subtype, ''), COALESCE(default_unit, '')
		FROM canonical_ingredients
		WHERE id = $1
	`, canonicalIngredientID).Scan(
		&preview.CanonicalIngredient.ID,
		&preview.CanonicalIngredient.Name,
		&preview.CanonicalIngredient.Category,
		&preview.CanonicalIngredient.Subtype,
		&preview.CanonicalIngredient.DefaultUnit,
	); err != nil {
		if err == sql.ErrNoRows {
			return CanonicalAliasProposalPreview{}, fmt.Errorf("postgres canonical alias proposal: canonical ingredient %q not found", canonicalIngredientID)
		}
		return CanonicalAliasProposalPreview{}, fmt.Errorf("postgres canonical alias proposal: load canonical ingredient: %w", err)
	}

	preview.AlreadyCanonical = canonicaltext.NormalizeText(preview.CanonicalIngredient.Name) == normalizedAlias ||
		canonicaltext.NormalizeText(preview.CanonicalIngredient.ID) == normalizedAlias

	var existingStatus string
	err := db.QueryRowContext(ctx, `
		SELECT status
		FROM canonical_ingredient_aliases
		WHERE canonical_ingredient_id = $1
		  AND normalized_alias = $2
	`, canonicalIngredientID, normalizedAlias).Scan(&existingStatus)
	if err != nil && err != sql.ErrNoRows {
		return CanonicalAliasProposalPreview{}, fmt.Errorf("postgres canonical alias proposal: inspect existing alias: %w", err)
	}
	if err == nil {
		preview.ExistingStatus = existingStatus
	}

	var conflict catalog.CanonicalIngredient
	err = db.QueryRowContext(ctx, `
		SELECT ci.id, ci.name, COALESCE(ci.category, ''), COALESCE(ci.subtype, ''), COALESCE(ci.default_unit, '')
		FROM canonical_ingredient_aliases a
		JOIN canonical_ingredients ci ON ci.id = a.canonical_ingredient_id
		WHERE a.normalized_alias = $1
		  AND a.status = 'verified'
		  AND a.canonical_ingredient_id <> $2
		LIMIT 1
	`, normalizedAlias, canonicalIngredientID).Scan(
		&conflict.ID,
		&conflict.Name,
		&conflict.Category,
		&conflict.Subtype,
		&conflict.DefaultUnit,
	)
	if err != nil && err != sql.ErrNoRows {
		return CanonicalAliasProposalPreview{}, fmt.Errorf("postgres canonical alias proposal: inspect verified conflict: %w", err)
	}
	if err == nil {
		preview.VerifiedConflict = &conflict
	}

	return preview, nil
}

func SaveCanonicalAliasProposal(ctx context.Context, db *sql.DB, input CanonicalAliasProposalInput) (SavedCanonicalAliasProposal, error) {
	preview, err := PreviewCanonicalAliasProposal(ctx, db, input.CanonicalIngredientID, input.Alias)
	if err != nil {
		return SavedCanonicalAliasProposal{}, err
	}
	if preview.AlreadyCanonical {
		return SavedCanonicalAliasProposal{}, fmt.Errorf(
			"postgres canonical alias proposal: %q already resolves as canonical ingredient %s",
			preview.Alias,
			preview.CanonicalIngredient.ID,
		)
	}
	if preview.VerifiedConflict != nil {
		return SavedCanonicalAliasProposal{}, fmt.Errorf(
			"postgres canonical alias proposal: normalized alias %q is already verified for %s",
			preview.NormalizedAlias,
			preview.VerifiedConflict.ID,
		)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return SavedCanonicalAliasProposal{}, fmt.Errorf("postgres canonical alias proposal: begin: %w", err)
	}
	defer tx.Rollback()

	var saved SavedCanonicalAliasProposal
	if err := tx.QueryRowContext(ctx, `
		INSERT INTO canonical_ingredient_aliases (
			canonical_ingredient_id,
			alias,
			normalized_alias,
			status,
			confidence,
			decision_source,
			updated_at
		) VALUES ($1, $2, $3, 'suggested', $4, $5, NOW())
		ON CONFLICT (canonical_ingredient_id, normalized_alias)
		DO UPDATE SET
			alias = CASE
				WHEN canonical_ingredient_aliases.status = 'suggested' THEN EXCLUDED.alias
				ELSE canonical_ingredient_aliases.alias
			END,
			confidence = CASE
				WHEN canonical_ingredient_aliases.status = 'suggested' THEN GREATEST(
					COALESCE(canonical_ingredient_aliases.confidence, 0),
					COALESCE(EXCLUDED.confidence, 0)
				)
				ELSE canonical_ingredient_aliases.confidence
			END,
			decision_source = CASE
				WHEN canonical_ingredient_aliases.status = 'suggested' THEN EXCLUDED.decision_source
				ELSE canonical_ingredient_aliases.decision_source
			END,
			updated_at = NOW()
		RETURNING id, status
	`,
		strings.TrimSpace(input.CanonicalIngredientID),
		preview.Alias,
		preview.NormalizedAlias,
		input.Confidence,
		strings.TrimSpace(input.DecisionSource),
	).Scan(&saved.AliasID, &saved.Status); err != nil {
		return SavedCanonicalAliasProposal{}, fmt.Errorf("postgres canonical alias proposal: save alias: %w", err)
	}

	// Evidence insertion is idempotent for the same proposal payload. This is
	// important for controlled Railway jobs that may be restarted after deploys.
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO canonical_alias_evidence (
			alias_id,
			evidence_type,
			source_ref,
			source_text
		)
		SELECT $1, $2, NULLIF($3, ''), NULLIF($4, '')
		WHERE NOT EXISTS (
			SELECT 1
			FROM canonical_alias_evidence e
			WHERE e.alias_id = $1
			  AND e.evidence_type = $2
			  AND COALESCE(e.source_ref, '') = $3
			  AND COALESCE(e.source_text, '') = $4
		)
	`,
		saved.AliasID,
		strings.TrimSpace(input.EvidenceType),
		strings.TrimSpace(input.EvidenceSourceRef),
		strings.TrimSpace(input.EvidenceSourceText),
	); err != nil {
		return SavedCanonicalAliasProposal{}, fmt.Errorf("postgres canonical alias proposal: save evidence: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return SavedCanonicalAliasProposal{}, fmt.Errorf("postgres canonical alias proposal: commit: %w", err)
	}
	return saved, nil
}
