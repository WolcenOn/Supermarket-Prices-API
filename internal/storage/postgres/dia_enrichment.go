package postgres

import (
    "context"
    "database/sql"
    "fmt"
    "strings"
)

const diaNutritionSource = "dia_product_page"

// DIAEnrichmentCandidate is an already-imported DIA product whose public
// product-detail URL can be enriched without rediscovering product identity.
type DIAEnrichmentCandidate struct {
    ProductID           string `json:"productId"`
    ExternalID          string `json:"externalId"`
    Name                string `json:"name"`
    SourceURL           string `json:"sourceUrl"`
    ItemType            string `json:"itemType,omitempty"`
    AlreadyHasNutrition bool   `json:"alreadyHasNutrition"`
}

// DIAEnrichmentDiagnostics reports mutually exclusive reasons why imported DIA
// products are or are not selectable for nutrition enrichment. It is read-only
// and deliberately mirrors the candidate selector's eligibility rules.
type DIAEnrichmentDiagnostics struct {
    TotalProducts       int `json:"totalProducts"`
    MissingSourceURL    int `json:"missingSourceUrl"`
    IneligibleItemType  int `json:"ineligibleItemType"`
    AlreadyHasNutrition int `json:"alreadyHasNutrition"`
    Selectable          int `json:"selectable"`
}

// DIAEnrichmentCandidates returns a bounded, deterministic set of existing
// products. It never invents a product URL and excludes non-food grocery types.
func DIAEnrichmentCandidates(ctx context.Context, db *sql.DB, limit int, includeExisting bool) ([]DIAEnrichmentCandidate, error) {
    if db == nil {
        return nil, fmt.Errorf("postgres dia enrichment: database is required")
    }
    if limit <= 0 {
        return []DIAEnrichmentCandidate{}, nil
    }
    if limit > 25 {
        limit = 25
    }

    rows, err := db.QueryContext(ctx, `
        SELECT
            sp.id::text,
            sp.external_id,
            sp.name,
            sp.source_url,
            COALESCE(sp.item_type, ''),
            EXISTS (
                SELECT 1
                FROM product_nutrition pn
                WHERE pn.supermarket_product_id = sp.id
                  AND pn.source = $1
            ) AS already_has_nutrition
        FROM supermarket_products sp
        WHERE sp.supermarket_id = 'dia'
          AND NULLIF(BTRIM(sp.source_url), '') IS NOT NULL
          AND COALESCE(sp.item_type, '') IN ('food_ingredient', 'prepared_food', 'beverage')
          AND (
              $2::boolean
              OR NOT EXISTS (
                  SELECT 1
                  FROM product_nutrition pn2
                  WHERE pn2.supermarket_product_id = sp.id
                    AND pn2.source = $1
              )
          )
        ORDER BY sp.updated_at DESC, sp.external_id ASC
        LIMIT $3
    `, diaNutritionSource, includeExisting, limit)
    if err != nil {
        return nil, fmt.Errorf("postgres dia enrichment: list candidates: %w", err)
    }
    defer rows.Close()

    out := make([]DIAEnrichmentCandidate, 0, limit)
    for rows.Next() {
        var item DIAEnrichmentCandidate
        if err := rows.Scan(
            &item.ProductID,
            &item.ExternalID,
            &item.Name,
            &item.SourceURL,
            &item.ItemType,
            &item.AlreadyHasNutrition,
        ); err != nil {
            return nil, fmt.Errorf("postgres dia enrichment: scan candidate: %w", err)
        }
        item.SourceURL = strings.TrimSpace(item.SourceURL)
        out = append(out, item)
    }
    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("postgres dia enrichment: rows: %w", err)
    }
    return out, nil
}

// DIAEnrichmentSelectionDiagnostics counts every imported DIA product exactly
// once: missing URL, URL present but ineligible type, eligible but already
// enriched, or currently selectable. This makes an empty candidate set
// explainable without fetching any supermarket product pages.
func DIAEnrichmentSelectionDiagnostics(ctx context.Context, db *sql.DB) (DIAEnrichmentDiagnostics, error) {
    if db == nil {
        return DIAEnrichmentDiagnostics{}, fmt.Errorf("postgres dia enrichment diagnostics: database is required")
    }

    var out DIAEnrichmentDiagnostics
    err := db.QueryRowContext(ctx, `
        SELECT
            COUNT(*)::int,
            COUNT(*) FILTER (
                WHERE NULLIF(BTRIM(sp.source_url), '') IS NULL
            )::int AS missing_source_url,
            COUNT(*) FILTER (
                WHERE NULLIF(BTRIM(sp.source_url), '') IS NOT NULL
                  AND COALESCE(sp.item_type, '') NOT IN ('food_ingredient', 'prepared_food', 'beverage')
            )::int AS ineligible_item_type,
            COUNT(*) FILTER (
                WHERE NULLIF(BTRIM(sp.source_url), '') IS NOT NULL
                  AND COALESCE(sp.item_type, '') IN ('food_ingredient', 'prepared_food', 'beverage')
                  AND EXISTS (
                      SELECT 1
                      FROM product_nutrition pn
                      WHERE pn.supermarket_product_id = sp.id
                        AND pn.source = $1
                  )
            )::int AS already_has_nutrition,
            COUNT(*) FILTER (
                WHERE NULLIF(BTRIM(sp.source_url), '') IS NOT NULL
                  AND COALESCE(sp.item_type, '') IN ('food_ingredient', 'prepared_food', 'beverage')
                  AND NOT EXISTS (
                      SELECT 1
                      FROM product_nutrition pn
                      WHERE pn.supermarket_product_id = sp.id
                        AND pn.source = $1
                  )
            )::int AS selectable
        FROM supermarket_products sp
        WHERE sp.supermarket_id = 'dia'
    `, diaNutritionSource).Scan(
        &out.TotalProducts,
        &out.MissingSourceURL,
        &out.IneligibleItemType,
        &out.AlreadyHasNutrition,
        &out.Selectable,
    )
    if err != nil {
        return DIAEnrichmentDiagnostics{}, fmt.Errorf("postgres dia enrichment diagnostics: %w", err)
    }
    return out, nil
}
