package dia

import (
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
    if len(product.Promotions) != 1 {
        t.Fatalf("promotions = %d, want 1", len(product.Promotions))
    }
    if product.Promotions[0].Type != "club" || product.Promotions[0].Price != 1.03 {
        t.Fatalf("unexpected promotion: %#v", product.Promotions[0])
    }
}

func TestNormalizeRejectsInvalidProduct(t *testing.T) {
    _, err := Normalize(RawProduct{ExternalID: "x", RegularPrice: 1})
    if err == nil {
        t.Fatal("Normalize() expected error for empty name")
    }
}
