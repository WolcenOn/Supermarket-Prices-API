package dia

import "testing"

func TestNormalizeInfersPackageSizeFromName(t *testing.T) {
    product, err := Normalize(RawProduct{
        ExternalID:   "21415",
        Name:         "Arroz redondo SOS 1 Kg",
        RegularPrice: 1.88,
        PricePerUnit: 1.88,
        PriceUnit:    "KILO",
        PostalCode:   "28001",
        Available:    true,
    })
    if err != nil {
        t.Fatalf("Normalize() error = %v", err)
    }
    if product.PackageAmount != 1 || product.PackageUnit != "kg" {
        t.Fatalf("package = %v %q, want 1 kg", product.PackageAmount, product.PackageUnit)
    }
}

func TestNormalizeKeepsStructuredPackageSize(t *testing.T) {
    product, err := Normalize(RawProduct{
        ExternalID:    "structured",
        Name:          "Producto 2 x 125 g",
        PackageAmount: 999,
        PackageUnit:   "g",
        RegularPrice:  2.10,
        Available:     true,
    })
    if err != nil {
        t.Fatalf("Normalize() error = %v", err)
    }
    if product.PackageAmount != 999 || product.PackageUnit != "g" {
        t.Fatalf("structured package overwritten: %v %q", product.PackageAmount, product.PackageUnit)
    }
}
