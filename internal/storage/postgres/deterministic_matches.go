package postgres

import (
    "context"
    "database/sql"
    "fmt"
    "strings"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

// LoadDeterministicMatchProducts returns already-classified recipe-compatible
// products that are eligible for conservative deterministic product matching.
// It deliberately reads product metadata only; no prices or observations are
// required to decide canonical identity.
func LoadDeterministicMatchProducts(ctx context.Context, db *sql.DB, supermarketID, family string, limit int) ([]catalog.Product, error) {
    if db == nil {
        return nil, fmt.Errorf("postgres deterministic matches: database is required")
    }
    supermarketID = strings.TrimSpace(supermarketID)
    family = strings.TrimSpace(strings.ToLower(family))
    if supermarketID == "" {
        return nil, fmt.Errorf("postgres deterministic matches: supermarket is required")
    }
    if limit <= 0 {
        return []catalog.Product{}, nil
    }

    category := ""
    switch family {
    case "all", "":
    case "rice":
        category = "food.pantry.cereal.rice"
    case "milk":
        category = "food.dairy.milk"
    case "vegetables":
        category = "food.produce.vegetable"
    default:
        return nil, fmt.Errorf("postgres deterministic matches: unsupported family %q", family)
    }

    rows, err := db.QueryContext(ctx, `
        SELECT
            id::text,
            supermarket_id,
            external_id,
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
        WHERE supermarket_id = $1
          AND classification_status = 'classified'
          AND recipe_compatible = TRUE
          AND item_type = 'food_ingredient'
          AND ($2 = '' OR normalized_category = $2)
        ORDER BY normalized_category, name, external_id
        LIMIT $3
    `, supermarketID, category, limit)
    if err != nil {
        return nil, fmt.Errorf("postgres deterministic matches: list products: %w", err)
    }
    defer rows.Close()

    products := make([]catalog.Product, 0)
    for rows.Next() {
        var product catalog.Product
        if err := rows.Scan(
            &product.ID,
            &product.SupermarketID,
            &product.ExternalID,
            &product.Name,
            &product.Brand,
            &product.SourceCategoryID,
            &product.SourceCategoryName,
            &product.SourceCategoryPath,
            &product.ItemType,
            &product.NormalizedCategory,
            &product.RecipeCompatible,
            &product.ClassificationStatus,
            &product.ClassificationScore,
            &product.ClassificationSource,
        ); err != nil {
            return nil, fmt.Errorf("postgres deterministic matches: scan product: %w", err)
        }
        products = append(products, product)
    }
    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("postgres deterministic matches: iterate products: %w", err)
    }
    return products, nil
}

func CanonicalIngredientExists(ctx context.Context, db *sql.DB, canonicalIngredientID string) (bool, error) {
    if db == nil {
        return false, fmt.Errorf("postgres deterministic matches: database is required")
    }
    canonicalIngredientID = strings.TrimSpace(canonicalIngredientID)
    if canonicalIngredientID == "" {
        return false, nil
    }
    var exists bool
    if err := db.QueryRowContext(ctx, `
        SELECT EXISTS (
            SELECT 1
            FROM canonical_ingredients
            WHERE id = $1
        )
    `, canonicalIngredientID).Scan(&exists); err != nil {
        return false, fmt.Errorf("postgres deterministic matches: check canonical %q: %w", canonicalIngredientID, err)
    }
    return exists, nil
}
