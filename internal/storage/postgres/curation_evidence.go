package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/WolcenOn/Supermarket-Prices-API/internal/curation"
)

func (s *CatalogStore) CurationAliasContext(ctx context.Context, canonicalIngredientID, alias string) (curation.AliasContext, error) {
	if s == nil || s.db == nil {
		return curation.AliasContext{}, fmt.Errorf("postgres curation evidence: database is required")
	}

	preview, err := PreviewCanonicalAliasProposal(ctx, s.db, canonicalIngredientID, alias)
	if err != nil {
		return curation.AliasContext{}, err
	}
	result := curation.AliasContext{
		CanonicalIngredientID: preview.CanonicalIngredient.ID,
		CanonicalName:         preview.CanonicalIngredient.Name,
		AlreadyCanonical:      preview.AlreadyCanonical,
		ExistingStatus:        preview.ExistingStatus,
	}
	if preview.VerifiedConflict != nil {
		result.VerifiedConflictID = preview.VerifiedConflict.ID
	}
	return result, nil
}

func (s *CatalogStore) CurationProductEvidence(ctx context.Context, productID string) (curation.ProductEvidence, bool, error) {
	if s == nil || s.db == nil {
		return curation.ProductEvidence{}, false, fmt.Errorf("postgres curation evidence: database is required")
	}

	productID = strings.TrimSpace(productID)
	if productID == "" {
		return curation.ProductEvidence{}, false, nil
	}

	var item curation.ProductEvidence
	err := s.db.QueryRowContext(ctx, `
		SELECT
			id::text,
			name,
			COALESCE(item_type, ''),
			COALESCE(normalized_category, ''),
			recipe_compatible,
			COALESCE(classification_status, ''),
			COALESCE(classification_source, '')
		FROM supermarket_products
		WHERE id = $1::uuid
	`, productID).Scan(
		&item.ID,
		&item.Name,
		&item.ItemType,
		&item.NormalizedCategory,
		&item.RecipeCompatible,
		&item.ClassificationStatus,
		&item.ClassificationSource,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return curation.ProductEvidence{}, false, nil
		}
		return curation.ProductEvidence{}, false, fmt.Errorf("postgres curation evidence: load product %q: %w", productID, err)
	}
	return item, true, nil
}
