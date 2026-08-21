package dia

import (
    "fmt"
    "strings"
    "time"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

const supermarketID = "dia"

// RawProduct is the acquisition-layer shape for DIA.
// It is intentionally source-agnostic enough to be populated from structured
// public data or HTML parsing without leaking DIA-specific fields to consumers.
type RawProduct struct {
    ExternalID       string
    Name             string
    Brand            string
    PackageAmount    float64
    PackageUnit      string
    RegularPrice     float64
    PricePerUnit     float64
    PriceUnit        string
    PromotionalPrice float64
    PromotionLabel   string
    PromotionType    string
    DiscountPct      float64
    PostalCode       string
    VariableWeight   bool
    Available        bool
    ObservedAt       time.Time
}

func Normalize(raw RawProduct) (catalog.Product, error) {
    raw.Name = strings.TrimSpace(raw.Name)
    raw.ExternalID = strings.TrimSpace(raw.ExternalID)
    raw.PostalCode = strings.TrimSpace(raw.PostalCode)

    if raw.Name == "" {
        return catalog.Product{}, fmt.Errorf("dia: product name is required")
    }
    if raw.ExternalID == "" {
        return catalog.Product{}, fmt.Errorf("dia: external id is required")
    }
    if raw.RegularPrice <= 0 {
        return catalog.Product{}, fmt.Errorf("dia: regular price must be positive")
    }

    packageAmount := raw.PackageAmount
    packageUnit := catalog.NormalizePackageUnit(raw.PackageUnit)
    if packageAmount <= 0 || packageUnit == "" {
        if inferredAmount, inferredUnit, ok := catalog.InferPackageSize(raw.Name); ok {
            packageAmount = inferredAmount
            packageUnit = inferredUnit
        }
    }

    observedAt := raw.ObservedAt.UTC()
    if raw.ObservedAt.IsZero() {
        observedAt = time.Now().UTC()
    }

    product := catalog.Product{
        ID:             supermarketID + "-" + raw.ExternalID,
        SupermarketID:  supermarketID,
        ExternalID:     raw.ExternalID,
        Name:           raw.Name,
        Brand:          strings.TrimSpace(raw.Brand),
        PackageAmount:  packageAmount,
        PackageUnit:    packageUnit,
        Price:          raw.RegularPrice,
        PricePerUnit:   raw.PricePerUnit,
        PriceUnit:      catalog.NormalizePackageUnit(raw.PriceUnit),
        PostalCode:     raw.PostalCode,
        VariableWeight: raw.VariableWeight,
        Available:      raw.Available,
        ObservedAt:     observedAt,
    }

    if raw.PromotionLabel != "" || raw.PromotionalPrice > 0 {
        promotionType := strings.TrimSpace(strings.ToLower(raw.PromotionType))
        if promotionType == "" {
            promotionType = "promotion"
        }
        product.Promotions = append(product.Promotions, catalog.Promotion{
            Type:        promotionType,
            Label:       strings.TrimSpace(raw.PromotionLabel),
            Price:       raw.PromotionalPrice,
            DiscountPct: raw.DiscountPct,
        })
    }

    return product, nil
}
