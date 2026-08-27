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
    ExternalID         string
    Name               string
    Brand              string
    EAN                string
    SourceURL          string
    PackageAmount      float64
    PackageUnit        string
    RegularPrice       float64
    PricePerUnit       float64
    PriceUnit          string
    PromotionalPrice   float64
    PromotionLabel     string
    PromotionType      string
    DiscountPct        float64
    PostalCode         string
    VariableWeight     bool
    Available          bool
    ObservedAt         time.Time
    SourceCategoryID   string
    SourceCategoryName string
    SourceCategoryPath string
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

    priceUnit := catalog.NormalizePackageUnit(raw.PriceUnit)
    if packageAmount <= 0 || packageUnit == "" {
        if inferredAmount, inferredUnit, ok := inferApproximateUnitPackage(raw.Name, raw.RegularPrice, raw.PricePerUnit, priceUnit); ok {
            packageAmount = inferredAmount
            packageUnit = inferredUnit
        }
    }

    variableWeight := raw.VariableWeight || inferVariableWeight(raw.Name, raw.PricePerUnit, priceUnit, packageAmount, packageUnit)
    if variableWeight {
        // True weight-sold products (for example "granel") do not have a
        // meaningful package size. Approximate whole units are kept as
        // packages instead so checkout quotes round to whole pieces.
        packageAmount = 0
        packageUnit = ""
    }

    observedAt := raw.ObservedAt.UTC()
    if raw.ObservedAt.IsZero() {
        observedAt = time.Now().UTC()
    }

    product := catalog.Product{
        ID:                 supermarketID + "-" + raw.ExternalID,
        SupermarketID:      supermarketID,
        ExternalID:         raw.ExternalID,
        Name:               raw.Name,
        Brand:              strings.TrimSpace(raw.Brand),
        EAN:                strings.TrimSpace(raw.EAN),
        SourceURL:          strings.TrimSpace(raw.SourceURL),
        PackageAmount:      packageAmount,
        PackageUnit:        packageUnit,
        Price:              raw.RegularPrice,
        PricePerUnit:       raw.PricePerUnit,
        PriceUnit:          priceUnit,
        PostalCode:         raw.PostalCode,
        VariableWeight:     variableWeight,
        Available:          raw.Available,
        ObservedAt:         observedAt,
        SourceCategoryID:   strings.TrimSpace(raw.SourceCategoryID),
        SourceCategoryName: strings.TrimSpace(raw.SourceCategoryName),
        SourceCategoryPath: strings.TrimSpace(raw.SourceCategoryPath),
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

func inferVariableWeight(name string, pricePerUnit float64, priceUnit string, packageAmount float64, packageUnit string) bool {
    if pricePerUnit <= 0 || (priceUnit != "kg" && priceUnit != "g") {
        return false
    }

    lowerName := strings.ToLower(strings.TrimSpace(name))
    if strings.Contains(lowerName, "granel") {
        return true
    }

    // "unidad" and explicit approximate weights describe a whole piece whose
    // final weight may vary. They are not evidence that an arbitrary fraction
    // can be ordered by weight.
    if strings.Contains(lowerName, "unidad") || (strings.Contains(lowerName, "aprox") && packageAmount > 0 && packageUnit != "") {
        return false
    }

    return packageAmount <= 0 || packageUnit == ""
}

func inferApproximateUnitPackage(name string, regularPrice float64, pricePerUnit float64, priceUnit string) (float64, string, bool) {
    lowerName := strings.ToLower(strings.TrimSpace(name))
    if !strings.Contains(lowerName, "unidad") || regularPrice <= 0 || pricePerUnit <= 0 {
        return 0, "", false
    }
    if priceUnit != "kg" && priceUnit != "g" {
        return 0, "", false
    }

    amount := regularPrice / pricePerUnit
    if amount <= 0 {
        return 0, "", false
    }
    return amount, priceUnit, true
}
