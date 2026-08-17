package dia

import (
    "context"
    "errors"
    "testing"
)

type stubSource struct {
    products []RawProduct
    err      error
}

func (s stubSource) Search(context.Context, string, string) ([]RawProduct, error) {
    return s.products, s.err
}

func TestProviderSearchNormalizesProducts(t *testing.T) {
    provider := NewProvider(stubSource{products: []RawProduct{
        {
            ExternalID:   "sku-1",
            Name:         "Arroz largo Dia Arrozona 1 Kg",
            Brand:        "Dia Arrozona",
            PackageAmount: 1,
            PackageUnit:  "Kg",
            RegularPrice: 1.25,
            PricePerUnit: 1.25,
            PriceUnit:    "KILO",
            Available:    true,
        },
        {
            ExternalID:   "",
            Name:         "Producto inválido",
            RegularPrice: 1,
        },
    }})

    products, err := provider.Search(context.Background(), "arroz", "28001")
    if err != nil {
        t.Fatal(err)
    }
    if len(products) != 1 {
        t.Fatalf("expected 1 valid product, got %d", len(products))
    }
    if products[0].SupermarketID != "dia" {
        t.Fatalf("expected DIA product, got %q", products[0].SupermarketID)
    }
    if products[0].PostalCode != "28001" {
        t.Fatalf("expected postal code to be propagated, got %q", products[0].PostalCode)
    }
    if products[0].PriceUnit != "kg" {
        t.Fatalf("expected normalized kg unit, got %q", products[0].PriceUnit)
    }
}

func TestProviderSearchPropagatesSourceError(t *testing.T) {
    provider := NewProvider(stubSource{err: errors.New("boom")})
    _, err := provider.Search(context.Background(), "arroz", "28001")
    if err == nil {
        t.Fatal("expected error")
    }
}
