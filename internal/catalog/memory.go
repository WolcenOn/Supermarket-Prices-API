package catalog

import (
    "strings"
    "time"
)

type MemoryStore struct {
    products []Product
}

func NewMemoryStore(products []Product) *MemoryStore {
    return &MemoryStore{products: products}
}

func (m *MemoryStore) Supermarkets() []Supermarket {
    return []Supermarket{
        {ID: "carrefour", Name: "Carrefour"},
        {ID: "alcampo", Name: "Alcampo"},
        {ID: "dia", Name: "DIA"},
        {ID: "mercadona", Name: "Mercadona"},
    }
}

func (m *MemoryStore) Search(params SearchParams) []Product {
    query := strings.ToLower(strings.TrimSpace(params.Query))
    postalCode := strings.TrimSpace(params.PostalCode)

    out := make([]Product, 0)
    for _, product := range m.products {
        if query != "" && !strings.Contains(strings.ToLower(product.Name+" "+product.Brand), query) {
            continue
        }
        if postalCode != "" && product.PostalCode != "" && product.PostalCode != postalCode {
            continue
        }
        out = append(out, product)
    }
    return out
}

func SeedProducts() []Product {
    now := time.Now().UTC()
    return []Product{
        {
            ID: "demo-carrefour-arroz-1kg", SupermarketID: "carrefour", ExternalID: "demo-001",
            Name: "Arroz redondo", Brand: "Demo Carrefour", PackageAmount: 1, PackageUnit: "kg",
            Price: 1.29, PricePerUnit: 1.29, PriceUnit: "kg", PostalCode: "28001", ObservedAt: now,
        },
        {
            ID: "demo-alcampo-arroz-1kg", SupermarketID: "alcampo", ExternalID: "demo-002",
            Name: "Arroz redondo", Brand: "Demo Alcampo", PackageAmount: 1, PackageUnit: "kg",
            Price: 1.24, PricePerUnit: 1.24, PriceUnit: "kg", PostalCode: "28001", ObservedAt: now,
        },
    }
}
