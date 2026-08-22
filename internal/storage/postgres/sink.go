package postgres

import (
    "context"
    "database/sql"
    "fmt"
    "strings"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

// Sink persists normalized catalog products and immutable price observations.
// Product metadata is updated in place, while every import creates a new
// price_observations row so price history is preserved.
type Sink struct {
    db *sql.DB
}

func NewSink(db *sql.DB) *Sink {
    return &Sink{db: db}
}

func (s *Sink) SaveProducts(ctx context.Context, products []catalog.Product) error {
    if s == nil || s.db == nil {
        return fmt.Errorf("postgres sink: database is required")
    }
    if len(products) == 0 {
        return nil
    }

    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("postgres sink: begin transaction: %w", err)
    }
    defer tx.Rollback()

    for _, product := range products {
        if err := saveProduct(ctx, tx, product); err != nil {
            return err
        }
    }

    if err := tx.Commit(); err != nil {
        return fmt.Errorf("postgres sink: commit: %w", err)
    }
    return nil
}

func saveProduct(ctx context.Context, tx *sql.Tx, product catalog.Product) error {
    var productID string
    err := tx.QueryRowContext(ctx, `
        INSERT INTO supermarket_products (
            supermarket_id,
            external_id,
            name,
            brand,
            package_amount,
            package_unit,
            variable_weight,
            source_category_id,
            source_category_name,
            source_category_path,
            item_type,
            normalized_category,
            recipe_compatible,
            classification_status,
            classification_score,
            classification_source,
            updated_at
        ) VALUES (
            $1,
            $2,
            $3,
            NULLIF($4, ''),
            NULLIF($5::numeric, 0::numeric),
            NULLIF($6, ''),
            $7,
            NULLIF($8, ''),
            NULLIF($9, ''),
            NULLIF($10, ''),
            NULLIF($11, ''),
            NULLIF($12, ''),
            $13,
            NULLIF($14, ''),
            NULLIF($15::numeric, 0::numeric),
            NULLIF($16, ''),
            NOW()
        )
        ON CONFLICT (supermarket_id, external_id)
        DO UPDATE SET
            name = EXCLUDED.name,
            brand = EXCLUDED.brand,
            package_amount = EXCLUDED.package_amount,
            package_unit = EXCLUDED.package_unit,
            variable_weight = EXCLUDED.variable_weight,
            source_category_id = EXCLUDED.source_category_id,
            source_category_name = EXCLUDED.source_category_name,
            source_category_path = EXCLUDED.source_category_path,
            item_type = EXCLUDED.item_type,
            normalized_category = EXCLUDED.normalized_category,
            recipe_compatible = EXCLUDED.recipe_compatible,
            classification_status = EXCLUDED.classification_status,
            classification_score = EXCLUDED.classification_score,
            classification_source = EXCLUDED.classification_source,
            updated_at = NOW()
        RETURNING id::text
    `,
        product.SupermarketID,
        product.ExternalID,
        strings.TrimSpace(product.Name),
        strings.TrimSpace(product.Brand),
        product.PackageAmount,
        strings.TrimSpace(product.PackageUnit),
        product.VariableWeight,
        strings.TrimSpace(product.SourceCategoryID),
        strings.TrimSpace(product.SourceCategoryName),
        strings.TrimSpace(product.SourceCategoryPath),
        strings.TrimSpace(product.ItemType),
        strings.TrimSpace(product.NormalizedCategory),
        product.RecipeCompatible,
        strings.TrimSpace(product.ClassificationStatus),
        product.ClassificationScore,
        strings.TrimSpace(product.ClassificationSource),
    ).Scan(&productID)
    if err != nil {
        return fmt.Errorf("postgres sink: upsert %s/%s: %w", product.SupermarketID, product.ExternalID, err)
    }

    var observationID string
    err = tx.QueryRowContext(ctx, `
        INSERT INTO price_observations (
            supermarket_product_id,
            postal_code,
            price,
            price_per_unit,
            price_unit,
            currency,
            available,
            observed_at
        ) VALUES (
            $1::uuid,
            NULLIF($2, ''),
            $3::numeric,
            NULLIF($4::numeric, 0::numeric),
            NULLIF($5, ''),
            'EUR',
            $6,
            $7
        )
        RETURNING id::text
    `,
        productID,
        strings.TrimSpace(product.PostalCode),
        product.Price,
        product.PricePerUnit,
        strings.TrimSpace(product.PriceUnit),
        product.Available,
        product.ObservedAt,
    ).Scan(&observationID)
    if err != nil {
        return fmt.Errorf("postgres sink: insert observation %s/%s: %w", product.SupermarketID, product.ExternalID, err)
    }

    for _, promotion := range product.Promotions {
        if _, err := tx.ExecContext(ctx, `
            INSERT INTO price_promotions (
                price_observation_id,
                promotion_type,
                label,
                promotional_price,
                discount_pct
            ) VALUES (
                $1::uuid,
                $2,
                NULLIF($3, ''),
                NULLIF($4::numeric, 0::numeric),
                NULLIF($5::numeric, 0::numeric)
            )
        `,
            observationID,
            strings.TrimSpace(promotion.Type),
            strings.TrimSpace(promotion.Label),
            promotion.Price,
            promotion.DiscountPct,
        ); err != nil {
            return fmt.Errorf("postgres sink: insert promotion %s/%s: %w", product.SupermarketID, product.ExternalID, err)
        }
    }

    return nil
}
