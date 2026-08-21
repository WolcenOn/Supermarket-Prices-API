package catalog

import "testing"

func TestQuotePurchasePackagedConvertsUnitsAndRoundsPackages(t *testing.T) {
    quote, err := QuotePurchase(Product{
        Name:          "Arroz redondo SOS 1 Kg",
        PackageAmount: 1,
        PackageUnit:   "kg",
        Price:         1.88,
    }, 1600, "g")
    if err != nil {
        t.Fatalf("QuotePurchase() error = %v", err)
    }
    if quote.PackageCount != 2 {
        t.Fatalf("PackageCount = %d, want 2", quote.PackageCount)
    }
    if quote.PurchasedAmount != 2 || quote.PurchasedUnit != "kg" {
        t.Fatalf("unexpected purchased amount: %#v", quote)
    }
    if quote.WasteAmount != 0.4 {
        t.Fatalf("WasteAmount = %v, want 0.4", quote.WasteAmount)
    }
    if quote.TotalCost != 3.76 {
        t.Fatalf("TotalCost = %v, want 3.76", quote.TotalCost)
    }
}

func TestQuotePurchaseVariableWeight(t *testing.T) {
    quote, err := QuotePurchase(Product{
        Name:           "Tomate pera",
        VariableWeight: true,
        PricePerUnit:   2.40,
        PriceUnit:      "kg",
    }, 500, "g")
    if err != nil {
        t.Fatalf("QuotePurchase() error = %v", err)
    }
    if quote.PurchasedAmount != 0.5 || quote.PurchasedUnit != "kg" {
        t.Fatalf("unexpected purchased amount: %#v", quote)
    }
    if quote.TotalCost != 1.20 {
        t.Fatalf("TotalCost = %v, want 1.20", quote.TotalCost)
    }
}

func TestQuotePurchaseRejectsIncompatibleUnits(t *testing.T) {
    _, err := QuotePurchase(Product{PackageAmount: 1, PackageUnit: "l", Price: 1}, 500, "g")
    if err == nil {
        t.Fatal("QuotePurchase() expected incompatible-unit error")
    }
}
