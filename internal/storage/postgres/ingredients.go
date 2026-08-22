package postgres

import (
    "context"
    "fmt"
    "strings"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

func (s *CatalogStore) Ingredients(ctx context.Context) ([]catalog.CanonicalIngredient, error) {
    return s.searchIngredients(ctx, "")
}

func (s *CatalogStore) SearchIngredients(ctx context.Context, query string) ([]catalog.CanonicalIngredient, error) {
    return s.searchIngredients(ctx, strings.TrimSpace(query))
}

func (s *CatalogStore) searchIngredients(ctx context.Context, query string) ([]catalog.CanonicalIngredient, error) {
    if s == nil || s.db == nil {
        return nil, fmt.Errorf("postgres catalog: database is required")
    }

    rows, err := s.db.QueryContext(ctx, `
        SELECT id, name, COALESCE(category, ''), COALESCE(subtype, ''), COALESCE(default_unit, '')
        FROM canonical_ingredients
        WHERE $1 = ''
           OR LOWER(name) LIKE '%' || LOWER($1) || '%'
           OR LOWER(COALESCE(category, '')) LIKE '%' || LOWER($1) || '%'
           OR LOWER(COALESCE(subtype, '')) LIKE '%' || LOWER($1) || '%'
        ORDER BY category, name
        LIMIT 200
    `, query)
    if err != nil {
        return nil, fmt.Errorf("postgres catalog: search ingredients: %w", err)
    }
    defer rows.Close()

    items := make([]catalog.CanonicalIngredient, 0)
    for rows.Next() {
        var item catalog.CanonicalIngredient
        if err := rows.Scan(&item.ID, &item.Name, &item.Category, &item.Subtype, &item.DefaultUnit); err != nil {
            return nil, fmt.Errorf("postgres catalog: scan ingredient: %w", err)
        }
        items = append(items, item)
    }
    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("postgres catalog: iterate ingredients: %w", err)
    }
    return items, nil
}

func (s *CatalogStore) IngredientProducts(ctx context.Context, ingredientID, postalCode string) ([]catalog.IngredientProduct, error) {
    if s == nil || s.db == nil {
        return nil, fmt.Errorf("postgres catalog: database is required")
    }

    ingredientID = strings.TrimSpace(ingredientID)
    postalCode = strings.TrimSpace(postalCode)
    if ingredientID == "" {
        return []catalog.IngredientProduct{}, nil
    }

    rows, err := s.db.QueryContext(ctx, `
        SELECT
            sp.id::text,
            sp.supermarket_id,
            sp.external_id,
            sp.name,
            COALESCE(sp.brand, ''),
            COALESCE(sp.package_amount, 0),
            COALESCE(sp.package_unit, ''),
            sp.variable_weight,
            COALESCE(sp.source_category_id, ''),
            COALESCE(sp.source_category_name, ''),
            COALESCE(sp.source_category_path, ''),
            COALESCE(sp.item_type, ''),
            COALESCE(sp.normalized_category, ''),
            sp.recipe_compatible,
            COALESCE(sp.classification_status, ''),
            COALESCE(sp.classification_score, 0),
            COALESCE(sp.classification_source, ''),
            po.price,
            COALESCE(po.price_per_unit, 0),
            COALESCE(po.price_unit, ''),
            COALESCE(po.postal_code, ''),
            po.available,
            po.observed_at,
            ipm.match_status,
            COALESCE(ipm.match_score, 0),
            ipm.match_source
        FROM ingredient_product_matches ipm
        JOIN supermarket_products sp ON sp.id = ipm.supermarket_product_id
        JOIN LATERAL (
            SELECT observation.*
            FROM price_observations observation
            WHERE observation.supermarket_product_id = sp.id
              AND ($2 = '' OR observation.postal_code = $2 OR observation.postal_code IS NULL)
            ORDER BY
                CASE WHEN $2 <> '' AND observation.postal_code = $2 THEN 0 ELSE 1 END,
                observation.observed_at DESC
            LIMIT 1
        ) po ON TRUE
        WHERE ipm.canonical_ingredient_id = $1
          AND ipm.match_status IN ('automatic', 'confirmed')
          AND sp.supermarket_id IN ('dia', 'mercadona', 'lidl')
        ORDER BY po.price_per_unit NULLS LAST, po.price, sp.supermarket_id, sp.name
        LIMIT 200
    `, ingredientID, postalCode)
    if err != nil {
        return nil, fmt.Errorf("postgres catalog: ingredient products: %w", err)
    }
    defer rows.Close()

    items := make([]catalog.IngredientProduct, 0)
    for rows.Next() {
        var item catalog.IngredientProduct
        if err := rows.Scan(
            &item.Product.ID,
            &item.Product.SupermarketID,
            &item.Product.ExternalID,
            &item.Product.Name,
            &item.Product.Brand,
            &item.Product.PackageAmount,
            &item.Product.PackageUnit,
            &item.Product.VariableWeight,
            &item.Product.SourceCategoryID,
            &item.Product.SourceCategoryName,
            &item.Product.SourceCategoryPath,
            &item.Product.ItemType,
            &item.Product.NormalizedCategory,
            &item.Product.RecipeCompatible,
            &item.Product.ClassificationStatus,
            &item.Product.ClassificationScore,
            &item.Product.ClassificationSource,
            &item.Product.Price,
            &item.Product.PricePerUnit,
            &item.Product.PriceUnit,
            &item.Product.PostalCode,
            &item.Product.Available,
            &item.Product.ObservedAt,
            &item.MatchStatus,
            &item.MatchScore,
            &item.MatchSource,
        ); err != nil {
            return nil, fmt.Errorf("postgres catalog: scan ingredient product: %w", err)
        }
        items = append(items, item)
    }
    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("postgres catalog: iterate ingredient products: %w", err)
    }
    return items, nil
}
