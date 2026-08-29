package postgres

import (
    "context"
    "database/sql"
    "fmt"
    "strings"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

// CatalogStore serves the public catalog from PostgreSQL. Products are mutable
// metadata; prices remain immutable observations and this reader selects the
// newest observation relevant to the requested postal code.
type CatalogStore struct {
    db *sql.DB
}

func NewCatalogStore(db *sql.DB) *CatalogStore {
    return &CatalogStore{db: db}
}

func (s *CatalogStore) Supermarkets(ctx context.Context) ([]catalog.Supermarket, error) {
    if s == nil || s.db == nil {
        return nil, fmt.Errorf("postgres catalog: database is required")
    }

    rows, err := s.db.QueryContext(ctx, `
        SELECT id, name
        FROM supermarkets
        WHERE id IN ('dia', 'mercadona', 'lidl')
        ORDER BY CASE id
            WHEN 'dia' THEN 1
            WHEN 'mercadona' THEN 2
            WHEN 'lidl' THEN 3
            ELSE 99
        END
    `)
    if err != nil {
        return nil, fmt.Errorf("postgres catalog: list supermarkets: %w", err)
    }
    defer rows.Close()

    items := make([]catalog.Supermarket, 0, 3)
    for rows.Next() {
        var item catalog.Supermarket
        if err := rows.Scan(&item.ID, &item.Name); err != nil {
            return nil, fmt.Errorf("postgres catalog: scan supermarket: %w", err)
        }
        items = append(items, item)
    }
    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("postgres catalog: iterate supermarkets: %w", err)
    }
    return items, nil
}

func (s *CatalogStore) Search(ctx context.Context, params catalog.SearchParams) ([]catalog.Product, error) {
    if s == nil || s.db == nil {
        return nil, fmt.Errorf("postgres catalog: database is required")
    }

    query := strings.TrimSpace(params.Query)
    postalCode := strings.TrimSpace(params.PostalCode)
    scope, ok := catalog.NormalizeSearchScope(params.Scope)
    if query == "" || !ok {
        return []catalog.Product{}, nil
    }

    rows, err := s.db.QueryContext(ctx, `
        SELECT
            sp.id::text,
            sp.supermarket_id,
            sp.external_id,
            sp.name,
            COALESCE(sp.brand, ''),
            COALESCE(sp.ean, ''),
            COALESCE(sp.source_url, ''),
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
            po.id::text,
            po.price,
            COALESCE(po.price_per_unit, 0),
            COALESCE(po.price_unit, ''),
            COALESCE(po.postal_code, ''),
            po.available,
            po.observed_at
        FROM supermarket_products sp
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
        WHERE sp.supermarket_id IN ('dia', 'mercadona', 'lidl')
          AND LOWER(sp.name || ' ' || COALESCE(sp.brand, '')) LIKE '%' || LOWER($1) || '%'
          AND (
              $3 = 'all'
              OR (
                  $3 = 'non_food'
                  AND (
                      COALESCE(sp.item_type, '') IN ('non_food', 'household')
                      OR COALESCE(sp.normalized_category, '') IN ('non_food', 'household')
                  )
              )
              OR (
                  $3 = 'food'
                  AND NOT (
                      COALESCE(sp.item_type, '') IN ('non_food', 'household')
                      OR COALESCE(sp.normalized_category, '') IN ('non_food', 'household')
                  )
              )
          )
        ORDER BY sp.supermarket_id, sp.name
        LIMIT 100
    `, query, postalCode, scope)
    if err != nil {
        return nil, fmt.Errorf("postgres catalog: search products: %w", err)
    }

    type resultRow struct {
        product       catalog.Product
        observationID string
    }
    results := make([]resultRow, 0)
    for rows.Next() {
        var item resultRow
        if err := rows.Scan(
            &item.product.ID,
            &item.product.SupermarketID,
            &item.product.ExternalID,
            &item.product.Name,
            &item.product.Brand,
            &item.product.EAN,
            &item.product.SourceURL,
            &item.product.PackageAmount,
            &item.product.PackageUnit,
            &item.product.VariableWeight,
            &item.product.SourceCategoryID,
            &item.product.SourceCategoryName,
            &item.product.SourceCategoryPath,
            &item.product.ItemType,
            &item.product.NormalizedCategory,
            &item.product.RecipeCompatible,
            &item.product.ClassificationStatus,
            &item.product.ClassificationScore,
            &item.product.ClassificationSource,
            &item.observationID,
            &item.product.Price,
            &item.product.PricePerUnit,
            &item.product.PriceUnit,
            &item.product.PostalCode,
            &item.product.Available,
            &item.product.ObservedAt,
        ); err != nil {
            rows.Close()
            return nil, fmt.Errorf("postgres catalog: scan product: %w", err)
        }
        results = append(results, item)
    }
    if err := rows.Err(); err != nil {
        rows.Close()
        return nil, fmt.Errorf("postgres catalog: iterate products: %w", err)
    }
    rows.Close()

    products := make([]catalog.Product, 0, len(results))
    for _, item := range results {
        promotions, err := s.promotions(ctx, item.observationID)
        if err != nil {
            return nil, err
        }
        item.product.Promotions = promotions
        products = append(products, item.product)
    }
    return products, nil
}

func (s *CatalogStore) promotions(ctx context.Context, observationID string) ([]catalog.Promotion, error) {
    rows, err := s.db.QueryContext(ctx, `
        SELECT
            promotion_type,
            COALESCE(label, ''),
            COALESCE(promotional_price, 0),
            COALESCE(discount_pct, 0)
        FROM price_promotions
        WHERE price_observation_id = $1::uuid
        ORDER BY created_at, id
    `, observationID)
    if err != nil {
        return nil, fmt.Errorf("postgres catalog: list promotions: %w", err)
    }
    defer rows.Close()

    promotions := make([]catalog.Promotion, 0)
    for rows.Next() {
        var promotion catalog.Promotion
        if err := rows.Scan(&promotion.Type, &promotion.Label, &promotion.Price, &promotion.DiscountPct); err != nil {
            return nil, fmt.Errorf("postgres catalog: scan promotion: %w", err)
        }
        promotions = append(promotions, promotion)
    }
    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("postgres catalog: iterate promotions: %w", err)
    }
    return promotions, nil
}
