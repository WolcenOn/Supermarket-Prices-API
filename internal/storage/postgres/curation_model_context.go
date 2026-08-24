package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/WolcenOn/Supermarket-Prices-API/internal/curation"
)

// CurationModelProductContext loads trusted source taxonomy and our own
// deterministic classification for a product before any model call.
func (s *CatalogStore) CurationModelProductContext(ctx context.Context, productID string) (curation.ModelProductContext, bool, error) {
	if s == nil || s.db == nil {
		return curation.ModelProductContext{}, false, fmt.Errorf("postgres curation model context: database is required")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return curation.ModelProductContext{}, false, nil
	}

	var item curation.ModelProductContext
	err := s.db.QueryRowContext(ctx, `
		SELECT
			id::text,
			supermarket_id,
			name,
			COALESCE(brand, ''),
			COALESCE(source_category_id, ''),
			COALESCE(source_category_name, ''),
			COALESCE(source_category_path, ''),
			COALESCE(item_type, ''),
			COALESCE(normalized_category, ''),
			recipe_compatible,
			COALESCE(classification_status, ''),
			COALESCE(classification_score, 0),
			COALESCE(classification_source, '')
		FROM supermarket_products
		WHERE id = $1::uuid
	`, productID).Scan(
		&item.ID,
		&item.SupermarketID,
		&item.Name,
		&item.Brand,
		&item.SourceCategoryID,
		&item.SourceCategoryName,
		&item.SourceCategoryPath,
		&item.ItemType,
		&item.NormalizedCategory,
		&item.RecipeCompatible,
		&item.ClassificationStatus,
		&item.ClassificationScore,
		&item.ClassificationSource,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return curation.ModelProductContext{}, false, nil
		}
		return curation.ModelProductContext{}, false, fmt.Errorf("postgres curation model context: load product %q: %w", productID, err)
	}
	return item, true, nil
}
