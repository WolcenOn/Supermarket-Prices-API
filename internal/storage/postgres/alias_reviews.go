package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

type CanonicalAliasReviewPreview struct {
	Alias               catalog.CanonicalIngredientAlias `json:"alias"`
	CanonicalIngredient catalog.CanonicalIngredient      `json:"canonicalIngredient"`
	EvidenceCount       int                              `json:"evidenceCount"`
	EvidenceTypes       []string                         `json:"evidenceTypes"`
	VerifiedConflict    *catalog.CanonicalIngredient     `json:"verifiedConflict,omitempty"`
}

type CanonicalAliasReviewInput struct {
	AliasID        int64
	Decision       string
	DecisionSource string
	Note           string
}

type SavedCanonicalAliasReview struct {
	DecisionID int64  `json:"decisionId"`
	AliasID    int64  `json:"aliasId"`
	Status     string `json:"status"`
}

func PreviewCanonicalAliasReview(ctx context.Context, db *sql.DB, aliasID int64) (CanonicalAliasReviewPreview, error) {
	if db == nil {
		return CanonicalAliasReviewPreview{}, fmt.Errorf("postgres canonical alias review: database is required")
	}
	if aliasID <= 0 {
		return CanonicalAliasReviewPreview{}, fmt.Errorf("postgres canonical alias review: alias id must be positive")
	}

	preview := CanonicalAliasReviewPreview{EvidenceTypes: []string{}}
	var confidence sql.NullFloat64
	var note sql.NullString
	if err := db.QueryRowContext(ctx, `
		SELECT
			a.id,
			a.canonical_ingredient_id,
			a.alias,
			a.normalized_alias,
			a.status,
			a.confidence,
			a.decision_source,
			a.verification_note,
			ci.id,
			ci.name,
			COALESCE(ci.category, ''),
			COALESCE(ci.subtype, ''),
			COALESCE(ci.default_unit, '')
		FROM canonical_ingredient_aliases a
		JOIN canonical_ingredients ci ON ci.id = a.canonical_ingredient_id
		WHERE a.id = $1
	`, aliasID).Scan(
		&preview.Alias.ID,
		&preview.Alias.CanonicalIngredientID,
		&preview.Alias.Alias,
		&preview.Alias.NormalizedAlias,
		&preview.Alias.Status,
		&confidence,
		&preview.Alias.DecisionSource,
		&note,
		&preview.CanonicalIngredient.ID,
		&preview.CanonicalIngredient.Name,
		&preview.CanonicalIngredient.Category,
		&preview.CanonicalIngredient.Subtype,
		&preview.CanonicalIngredient.DefaultUnit,
	); err != nil {
		if err == sql.ErrNoRows {
			return CanonicalAliasReviewPreview{}, fmt.Errorf("postgres canonical alias review: alias %d not found", aliasID)
		}
		return CanonicalAliasReviewPreview{}, fmt.Errorf("postgres canonical alias review: load alias: %w", err)
	}
	if confidence.Valid {
		preview.Alias.Confidence = confidence.Float64
	}
	if note.Valid {
		preview.Alias.VerificationNote = note.String
	}

	rows, err := db.QueryContext(ctx, `
		SELECT evidence_type, COUNT(*)
		FROM canonical_alias_evidence
		WHERE alias_id = $1
		GROUP BY evidence_type
		ORDER BY evidence_type
	`, aliasID)
	if err != nil {
		return CanonicalAliasReviewPreview{}, fmt.Errorf("postgres canonical alias review: load evidence: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var evidenceType string
		var count int
		if err := rows.Scan(&evidenceType, &count); err != nil {
			return CanonicalAliasReviewPreview{}, fmt.Errorf("postgres canonical alias review: scan evidence: %w", err)
		}
		preview.EvidenceTypes = append(preview.EvidenceTypes, evidenceType)
		preview.EvidenceCount += count
	}
	if err := rows.Err(); err != nil {
		return CanonicalAliasReviewPreview{}, fmt.Errorf("postgres canonical alias review: iterate evidence: %w", err)
	}

	var conflict catalog.CanonicalIngredient
	err = db.QueryRowContext(ctx, `
		SELECT ci.id, ci.name, COALESCE(ci.category, ''), COALESCE(ci.subtype, ''), COALESCE(ci.default_unit, '')
		FROM canonical_ingredient_aliases a
		JOIN canonical_ingredients ci ON ci.id = a.canonical_ingredient_id
		WHERE a.normalized_alias = $1
		  AND a.status = 'verified'
		  AND a.id <> $2
		LIMIT 1
	`, preview.Alias.NormalizedAlias, aliasID).Scan(
		&conflict.ID,
		&conflict.Name,
		&conflict.Category,
		&conflict.Subtype,
		&conflict.DefaultUnit,
	)
	if err != nil && err != sql.ErrNoRows {
		return CanonicalAliasReviewPreview{}, fmt.Errorf("postgres canonical alias review: inspect verified conflict: %w", err)
	}
	if err == nil {
		preview.VerifiedConflict = &conflict
	}

	return preview, nil
}

func SaveCanonicalAliasReview(ctx context.Context, db *sql.DB, input CanonicalAliasReviewInput) (SavedCanonicalAliasReview, error) {
	if db == nil {
		return SavedCanonicalAliasReview{}, fmt.Errorf("postgres canonical alias review: database is required")
	}
	decision := strings.TrimSpace(strings.ToLower(input.Decision))
	decisionSource := strings.TrimSpace(input.DecisionSource)
	note := strings.TrimSpace(input.Note)
	if input.AliasID <= 0 {
		return SavedCanonicalAliasReview{}, fmt.Errorf("postgres canonical alias review: alias id must be positive")
	}
	if decision != "verified" && decision != "rejected" {
		return SavedCanonicalAliasReview{}, fmt.Errorf("postgres canonical alias review: decision must be verified or rejected")
	}
	if decisionSource == "" || note == "" {
		return SavedCanonicalAliasReview{}, fmt.Errorf("postgres canonical alias review: decision source and note are required")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return SavedCanonicalAliasReview{}, fmt.Errorf("postgres canonical alias review: begin: %w", err)
	}
	defer tx.Rollback()

	var currentStatus, normalizedAlias string
	if err := tx.QueryRowContext(ctx, `
		SELECT status, normalized_alias
		FROM canonical_ingredient_aliases
		WHERE id = $1
		FOR UPDATE
	`, input.AliasID).Scan(&currentStatus, &normalizedAlias); err != nil {
		if err == sql.ErrNoRows {
			return SavedCanonicalAliasReview{}, fmt.Errorf("postgres canonical alias review: alias %d not found", input.AliasID)
		}
		return SavedCanonicalAliasReview{}, fmt.Errorf("postgres canonical alias review: lock alias: %w", err)
	}
	if currentStatus != "suggested" {
		return SavedCanonicalAliasReview{}, fmt.Errorf("postgres canonical alias review: alias %d is %s, expected suggested", input.AliasID, currentStatus)
	}

	if decision == "verified" {
		var evidenceCount int
		if err := tx.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM canonical_alias_evidence WHERE alias_id = $1
		`, input.AliasID).Scan(&evidenceCount); err != nil {
			return SavedCanonicalAliasReview{}, fmt.Errorf("postgres canonical alias review: count evidence: %w", err)
		}
		if evidenceCount == 0 {
			return SavedCanonicalAliasReview{}, fmt.Errorf("postgres canonical alias review: verified aliases require at least one evidence record")
		}

		var conflictingID int64
		err := tx.QueryRowContext(ctx, `
			SELECT id
			FROM canonical_ingredient_aliases
			WHERE normalized_alias = $1
			  AND status = 'verified'
			  AND id <> $2
			LIMIT 1
		`, normalizedAlias, input.AliasID).Scan(&conflictingID)
		if err != nil && err != sql.ErrNoRows {
			return SavedCanonicalAliasReview{}, fmt.Errorf("postgres canonical alias review: inspect verified conflict: %w", err)
		}
		if err == nil {
			return SavedCanonicalAliasReview{}, fmt.Errorf("postgres canonical alias review: normalized alias is already verified by alias %d", conflictingID)
		}
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE canonical_ingredient_aliases
		SET status = $2,
		    decision_source = $3,
		    verification_note = $4,
		    updated_at = NOW()
		WHERE id = $1
	`, input.AliasID, decision, decisionSource, note); err != nil {
		return SavedCanonicalAliasReview{}, fmt.Errorf("postgres canonical alias review: update alias: %w", err)
	}

	var saved SavedCanonicalAliasReview
	saved.AliasID = input.AliasID
	saved.Status = decision
	if err := tx.QueryRowContext(ctx, `
		INSERT INTO canonical_alias_decisions (
			alias_id,
			from_status,
			to_status,
			decision_source,
			note
		) VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, input.AliasID, currentStatus, decision, decisionSource, note).Scan(&saved.DecisionID); err != nil {
		return SavedCanonicalAliasReview{}, fmt.Errorf("postgres canonical alias review: save decision: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return SavedCanonicalAliasReview{}, fmt.Errorf("postgres canonical alias review: commit: %w", err)
	}
	return saved, nil
}
