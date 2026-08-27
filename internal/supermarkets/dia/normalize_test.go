package dia

import (
    "math"
    "testing"
    "time"
)

func TestNormalizeClubPromotion(t *testing.T) {
    observed := time.Date(2026, 8, 17, 8, 0, 0, 0, time.UTC)

    product, err := Normalize(RawProduct{
        ExternalID:       "12345",
        Name:             "Hummus clásico Dia Al Punto 220 g",
        Brand:            "Dia Al Punto",
        EAN:              "8410830001016",
        SourceURL:        "https://www.dia.es/algo/p/12345",
        PackageAmount:    220,
        PackageUnit:      "g",
        RegularPrice:     1.29,
        PricePerUnit:     4.69,
        PriceUnit:        "KILO",
        PromotionalPrice: 1.03,
        PromotionLabel:   "Oferta CLUB Dia",
        PromotionType:    "club",
        DiscountPct:      20,
        PostalCode:       "28001",
        Available:        true,
        ObservedAt:       observed,
    })
    if err != nil {
        t.Fatalf("Normalize() error = %v", err)
    }

    if product.ID != "dia-12345" || product.SupermarketID != "dia" {
        t.Fatalf("unexpected product identity: %#v", product)
    }
    if product.EAN != "8410830001016" || product.SourceURL != "https://www.dia.es/algo/p/12345" {
        t.Fatalf("unexpected identifiers: ean=%q source=%q", product.EAN, product.SourceURL)
    }
    if product.PriceUnit != "kg" {
        t.Fatalf("PriceUnit = %q, want kg", product.PriceUnit)
    }
    if product.VariableWeight {
        t.Fatal("fixed packaged product must not be variable weight")
    }
    if len(product.Promotions) != 1 {
        t.Fatalf("promotions = %d, want 1", len(product.Promotions))
    }
    if product.Promotions[0].Type != "club" || product.Promotions[0].Price != 1.03 {
        t.Fatalf("unexpected promotion: %#v", product.Promotions[0])
    }
}

func TestNormalizeInfersTrueVariableWeightProduce(t *testing.T) {
    product, err := Normalize(RawProduct{
        ExternalID:   "produce-granel",
        Name:         "Tomate pera granel 900 g aprox.",
        RegularPrice: 2.06,
        PricePerUnit: 2.29,
        PriceUnit:    "kg",
        Available:    true,
    })
    if err != nil {
        t.Fatalf("Normalize() error = %v", err)
    }
    if !product.VariableWeight {
        t.Fatal("granel product must be variable weight")
    }
    if product.PackageAmount != 0 || product.PackageUnit != "" {
        t.Fatalf("variable-weight product must not expose fixed package size: %#v", product)
    }
}

func TestNormalizeKeepsApproximateWholeUnitsAsPackages(t *testing.T) {
    tests := []struct {
        name          string
        product       string
        price         float64
        pricePerKg    float64
        wantAmount    float64
        wantUnit      string
    }{
        {"explicit approximate unit", "Calabacín unidad 650 g aprox.", 1.10, 1.69, 650, "g"},
        {"explicit approximate piece", "Calabaza 1.6 Kg aprox.", 3.18, 1.99, 1.6, "kg"},
        {"unit inferred from price ratio", "Berenjena unidad", 0.79, 2.63, 0.79 / 2.63, "kg"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            product, err := Normalize(RawProduct{
                ExternalID:   "produce-unit",
                Name:         tt.product,
                RegularPrice: tt.price,
                PricePerUnit: tt.pricePerKg,
                PriceUnit:    "kg",
                Available:    true,
            })
            if err != nil {
                t.Fatalf("Normalize() error = %v", err)
            }
            if product.VariableWeight {
                t.Fatalf("%q is a whole approximate unit, not granel", tt.product)
            }
            if product.PackageUnit != tt.wantUnit || math.Abs(product.PackageAmount-tt.wantAmount) > 0.001 {
                t.Fatalf("unexpected approximate package for %q: %#v", tt.product, product)
            }
        })
    }
}

func TestNormalizeKeepsFixedProducePackaged(t *testing.T) {
    product, err := Normalize(RawProduct{
        ExternalID:   "produce-2",
        Name:         "Tomate pera bandeja 500 g",
        RegularPrice: 1.39,
        PricePerUnit: 2.78,
        PriceUnit:    "kg",
        Available:    true,
    })
    if err != nil {
        t.Fatalf("Normalize() error = %v", err)
    }
    if product.VariableWeight {
        t.Fatal("fixed tray must not be variable weight")
    }
    if product.PackageAmount != 500 || product.PackageUnit != "g" {
        t.Fatalf("unexpected package size: %#v", product)
    }
}

func TestNormalizeRejectsInvalidProduct(t *testing.T) {
    _, err := Normalize(RawProduct{ExternalID: "x", RegularPrice: 1})
    if err == nil {
        t.Fatal("Normalize() expected error for empty name")
    }
}
