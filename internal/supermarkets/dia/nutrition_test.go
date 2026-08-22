package dia

import (
    "testing"
    "time"
)

func TestProductNutritionPreservesMissingAndZeroValues(t *testing.T) {
    zero := 0.0
    protein := 3.8
    observed := time.Date(2026, 8, 22, 18, 30, 0, 0, time.UTC)
    details := ProductDetails{
        ExternalID:       "303311",
        SourceURL:        "https://www.dia.es/congelados-y-helados/arroces-y-pasta/p/303311",
        DescriptionText:  "ARROZ CON MARISCO",
        NutritionSource:  "dia_product_page",
        Nutrition: &NutritionFacts{
            EnergyKcal: &zero,
            ProteinG:   &protein,
        },
    }

    item, err := details.ProductNutrition(observed)
    if err != nil {
        t.Fatal(err)
    }
    if item.SupermarketID != "dia" || item.ExternalID != "303311" || item.Source != "dia_product_page" {
        t.Fatalf("unexpected identity: %+v", item)
    }
    if item.EnergyKcal == nil || *item.EnergyKcal != 0 {
        t.Fatalf("published zero must remain present: %+v", item)
    }
    if item.FatG != nil {
        t.Fatalf("missing fat must remain absent: %+v", item)
    }
    if item.ProteinG == nil || *item.ProteinG != 3.8 {
        t.Fatalf("unexpected protein: %+v", item)
    }
    if !item.ObservedAt.Equal(observed) {
        t.Fatalf("unexpected observation time %v", item.ObservedAt)
    }
}

func TestProductNutritionRequiresStructuredNutrition(t *testing.T) {
    _, err := (ProductDetails{ExternalID: "303311"}).ProductNutrition(time.Time{})
    if err == nil {
        t.Fatal("expected missing structured nutrition to fail")
    }
}
