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
        {ID: "dia", Name: "DIA"},
        {ID: "mercadona", Name: "Mercadona"},
        {ID: "lidl", Name: "Lidl"},
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
            ID: "demo-dia-arroz-1kg", SupermarketID: "dia", ExternalID: "demo-dia-001",
            Name: "Arroz redondo", Brand: "Dia", PackageAmount: 1, PackageUnit: "kg",
            Price: 1.29, PricePerUnit: 1.29, PriceUnit: "kg", PostalCode: "28001",
            Available: true, ObservedAt: now,
        },
        {
            ID: "demo-mercadona-arroz-1kg", SupermarketID: "mercadona", ExternalID: "demo-mercadona-001",
            Name: "Arroz redondo", Brand: "Hacendado", PackageAmount: 1, PackageUnit: "kg",
            Price: 1.35, PricePerUnit: 1.35, PriceUnit: "kg", PostalCode: "28001",
            Available: true, ObservedAt: now,
        },
        {
            ID: "demo-lidl-arroz-1kg", SupermarketID: "lidl", ExternalID: "demo-lidl-001",
            Name: "Arroz redondo", Brand: "Demo Lidl", PackageAmount: 1, PackageUnit: "kg",
            Price: 1.25, PricePerUnit: 1.25, PriceUnit: "kg", PostalCode: "28001",
            Available: true, ObservedAt: now,
        },
    }
}
