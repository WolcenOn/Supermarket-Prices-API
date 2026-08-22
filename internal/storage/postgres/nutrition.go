package postgres

import (
    "context"
    "database/sql"
    "fmt"
    "strings"
    "time"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

// SaveProductNutrition upserts one sourced nutrition snapshot for an existing
// commercial product. The product must already exist in supermarket_products;
// enrichment never creates a product identity on its own.
func SaveProductNutrition(ctx context.Context, db *sql.DB, nutrition catalog.ProductNutrition) (catalog.ProductNutrition, error) {
    if db == nil {
        return catalog.ProductNutrition{}, fmt.Errorf("postgres nutrition: database is required")
    }

    nutrition.SupermarketID = strings.TrimSpace(nutrition.SupermarketID)
    nutrition.ExternalID = strings.TrimSpace(nutrition.ExternalID)
    nutrition.Source = strings.TrimSpace(nutrition.Source)
    nutrition.SourceURL = strings.TrimSpace(nutrition.SourceURL)
    nutrition.EAN = strings.TrimSpace(nutrition.EAN)
    nutrition.BasisUnit = strings.TrimSpace(strings.ToLower(nutrition.BasisUnit))

    if nutrition.SupermarketID == "" || nutrition.ExternalID == "" {
        return catalog.ProductNutrition{}, fmt.Errorf("postgres nutrition: supermarket and external id are required")
    }
    if nutrition.Source == "" {
        return catalog.ProductNutrition{}, fmt.Errorf("postgres nutrition: source is required")
    }
    if nutrition.ObservedAt.IsZero() {
        nutrition.ObservedAt = time.Now().UTC()
    } else {
        nutrition.ObservedAt = nutrition.ObservedAt.UTC()
    }

    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return catalog.ProductNutrition{}, fmt.Errorf("postgres nutrition: begin transaction: %w", err)
    }
    defer tx.Rollback()

    var productID string
    err = tx.QueryRowContext(ctx, `
        UPDATE supermarket_products
        SET
            ean = COALESCE(NULLIF($3, ''), ean),
            source_url = COALESCE(NULLIF($4, ''), source_url),
            updated_at = NOW()
        WHERE supermarket_id = $1
          AND external_id = $2
        RETURNING id::text
    `,
        nutrition.SupermarketID,
        nutrition.ExternalID,
        nutrition.EAN,
        nutrition.SourceURL,
    ).Scan(&productID)
    if err == sql.ErrNoRows {
        return catalog.ProductNutrition{}, fmt.Errorf("postgres nutrition: product %s/%s does not exist; import it before enrichment", nutrition.SupermarketID, nutrition.ExternalID)
    }
    if err != nil {
        return catalog.ProductNutrition{}, fmt.Errorf("postgres nutrition: resolve product %s/%s: %w", nutrition.SupermarketID, nutrition.ExternalID, err)
    }

    nutrition.ProductID = productID

    _, err = tx.ExecContext(ctx, `
        INSERT INTO product_nutrition (
            supermarket_product_id,
            source,
            source_url,
            observed_at,
            description_text,
            source_ingredients_block,
            ingredients_text,
            responsible_text,
            basis_amount,
            basis_unit,
            energy_kj,
            energy_kcal,
            fat_g,
            saturated_fat_g,
            carbohydrates_g,
            sugars_g,
            fiber_g,
            protein_g,
            salt_g,
            updated_at
        ) VALUES (
            $1::uuid,
            $2,
            NULLIF($3, ''),
            $4,
            NULLIF($5, ''),
            NULLIF($6, ''),
            NULLIF($7, ''),
            NULLIF($8, ''),
            $9::numeric,
            NULLIF($10, ''),
            $11::numeric,
            $12::numeric,
            $13::numeric,
            $14::numeric,
            $15::numeric,
            $16::numeric,
            $17::numeric,
            $18::numeric,
            $19::numeric,
            NOW()
        )
        ON CONFLICT (supermarket_product_id, source)
        DO UPDATE SET
            source_url = EXCLUDED.source_url,
            observed_at = EXCLUDED.observed_at,
            description_text = EXCLUDED.description_text,
            source_ingredients_block = EXCLUDED.source_ingredients_block,
            ingredients_text = EXCLUDED.ingredients_text,
            responsible_text = EXCLUDED.responsible_text,
            basis_amount = EXCLUDED.basis_amount,
            basis_unit = EXCLUDED.basis_unit,
            energy_kj = EXCLUDED.energy_kj,
            energy_kcal = EXCLUDED.energy_kcal,
            fat_g = EXCLUDED.fat_g,
            saturated_fat_g = EXCLUDED.saturated_fat_g,
            carbohydrates_g = EXCLUDED.carbohydrates_g,
            sugars_g = EXCLUDED.sugars_g,
            fiber_g = EXCLUDED.fiber_g,
            protein_g = EXCLUDED.protein_g,
            salt_g = EXCLUDED.salt_g,
            updated_at = NOW()
    `,
        nutrition.ProductID,
        nutrition.Source,
        nutrition.SourceURL,
        nutrition.ObservedAt,
        strings.TrimSpace(nutrition.DescriptionText),
        strings.TrimSpace(nutrition.SourceIngredientsBlock),
        strings.TrimSpace(nutrition.IngredientsText),
        strings.TrimSpace(nutrition.ResponsibleText),
        nullableNutritionValue(nutrition.BasisAmount),
        nutrition.BasisUnit,
        nullableNutritionValue(nutrition.EnergyKJ),
        nullableNutritionValue(nutrition.EnergyKcal),
        nullableNutritionValue(nutrition.FatG),
        nullableNutritionValue(nutrition.SaturatedFatG),
        nullableNutritionValue(nutrition.CarbohydratesG),
        nullableNutritionValue(nutrition.SugarsG),
        nullableNutritionValue(nutrition.FiberG),
        nullableNutritionValue(nutrition.ProteinG),
        nullableNutritionValue(nutrition.SaltG),
    )
    if err != nil {
        return catalog.ProductNutrition{}, fmt.Errorf("postgres nutrition: upsert %s/%s from %s: %w", nutrition.SupermarketID, nutrition.ExternalID, nutrition.Source, err)
    }

    if err := tx.Commit(); err != nil {
        return catalog.ProductNutrition{}, fmt.Errorf("postgres nutrition: commit: %w", err)
    }

    return LoadProductNutrition(ctx, db, nutrition.SupermarketID, nutrition.ExternalID, nutrition.Source)
}

// LoadProductNutrition returns one source-specific nutrition snapshot for an
// exact commercial product.
func LoadProductNutrition(ctx context.Context, db *sql.DB, supermarketID, externalID, source string) (catalog.ProductNutrition, error) {
    if db == nil {
        return catalog.ProductNutrition{}, fmt.Errorf("postgres nutrition: database is required")
    }

    supermarketID = strings.TrimSpace(supermarketID)
    externalID = strings.TrimSpace(externalID)
    source = strings.TrimSpace(source)

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

    err := db.QueryRowContext(ctx, `
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
        WHERE sp.supermarket_id = $1
          AND sp.external_id = $2
          AND pn.source = $3
    `, supermarketID, externalID, source).Scan(
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
    )
    if err == sql.ErrNoRows {
        return catalog.ProductNutrition{}, fmt.Errorf("postgres nutrition: no nutrition for %s/%s from %s", supermarketID, externalID, source)
    }
    if err != nil {
        return catalog.ProductNutrition{}, fmt.Errorf("postgres nutrition: load %s/%s from %s: %w", supermarketID, externalID, source, err)
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
    return item, nil
}

func nullableNutritionValue(value *float64) any {
    if value == nil {
        return nil
    }
    return *value
}

func nutritionPointer(value sql.NullFloat64) *float64 {
    if !value.Valid {
        return nil
    }
    copy := value.Float64
    return &copy
}
