package postgres

import (
    "context"
    "database/sql"
    "fmt"
    "strings"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

// ProductNutrition returns every sourced nutrition snapshot currently stored
// for one exact commercial product. Sources remain separate so callers can see
// where each value came from instead of receiving an implicit merged record.
func (s *CatalogStore) ProductNutrition(ctx context.Context, productID string) ([]catalog.ProductNutrition, error) {
    if s == nil || s.db == nil {
        return nil, fmt.Errorf("postgres nutrition: database is required")
    }

    productID = strings.TrimSpace(productID)
    if productID == "" {
        return []catalog.ProductNutrition{}, nil
    }

    rows, err := s.db.QueryContext(ctx, `
        SELECT
            sp.id::text,
            sp.supermarket_id,
            sp.external_id,
            COALESCE(sp.ean, ''),
            pn.source,
            COALESCE(pn.source_url, ''),
            pn.observed_at,
            COALESCE(pn.description_text, ''),
            COALESCE(pn.source_ingredients_block, ''),
            COALESCE(pn.ingredients_text, ''),
            COALESCE(pn.responsible_text, ''),
            pn.basis_amount,
            COALESCE(pn.basis_unit, ''),
            pn.energy_kj,
            pn.energy_kcal,
            pn.fat_g,
            pn.saturated_fat_g,
            pn.carbohydrates_g,
            pn.sugars_g,
            pn.fiber_g,
            pn.protein_g,
            pn.salt_g
        FROM product_nutrition pn
        JOIN supermarket_products sp ON sp.id = pn.supermarket_product_id
        WHERE sp.id::text = $1
        ORDER BY pn.source
    `, productID)
    if err != nil {
        return nil, fmt.Errorf("postgres nutrition: list product %s: %w", productID, err)
    }
    defer rows.Close()

    items := make([]catalog.ProductNutrition, 0)
    for rows.Next() {
        var item catalog.ProductNutrition
        var basisAmount sql.NullFloat64
        var energyKJ sql.NullFloat64
        var energyKcal sql.NullFloat64
        var fatG sql.NullFloat64
        var saturatedFatG sql.NullFloat64
        var carbohydratesG sql.NullFloat64
        var sugarsG sql.NullFloat64
        var fiberG sql.NullFloat64
        var proteinG sql.NullFloat64
        var saltG sql.NullFloat64

        if err := rows.Scan(
            &item.ProductID,
            &item.SupermarketID,
            &item.ExternalID,
            &item.EAN,
            &item.Source,
            &item.SourceURL,
            &item.ObservedAt,
            &item.DescriptionText,
            &item.SourceIngredientsBlock,
            &item.IngredientsText,
            &item.ResponsibleText,
            &basisAmount,
            &item.BasisUnit,
            &energyKJ,
            &energyKcal,
            &fatG,
            &saturatedFatG,
            &carbohydratesG,
            &sugarsG,
            &fiberG,
            &proteinG,
            &saltG,
        ); err != nil {
            return nil, fmt.Errorf("postgres nutrition: scan product %s: %w", productID, err)
        }

        item.BasisAmount = nutritionPointer(basisAmount)
        item.EnergyKJ = nutritionPointer(energyKJ)
        item.EnergyKcal = nutritionPointer(energyKcal)
        item.FatG = nutritionPointer(fatG)
        item.SaturatedFatG = nutritionPointer(saturatedFatG)
        item.CarbohydratesG = nutritionPointer(carbohydratesG)
        item.SugarsG = nutritionPointer(sugarsG)
        item.FiberG = nutritionPointer(fiberG)
        item.ProteinG = nutritionPointer(proteinG)
        item.SaltG = nutritionPointer(saltG)
        item.ObservedAt = item.ObservedAt.UTC()
        items = append(items, item)
    }
    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("postgres nutrition: iterate product %s: %w", productID, err)
    }
    return items, nil
}
