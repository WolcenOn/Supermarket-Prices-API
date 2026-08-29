package catalog

import (
    "context"
    "strings"
    "time"
    "unicode"
)

type MemoryStore struct {
    products []Product
}

func NewMemoryStore(products []Product) *MemoryStore {
    return &MemoryStore{products: products}
}

func (m *MemoryStore) Supermarkets(_ context.Context) ([]Supermarket, error) {
    return []Supermarket{
        {ID: "dia", Name: "DIA"},
        {ID: "mercadona", Name: "Mercadona"},
        {ID: "lidl", Name: "Lidl"},
    }, nil
}

// NormalizeSearchText turns user/product text into accent-insensitive lowercase
// tokens. Product search uses whole tokens for meaningful terms so a query such
// as "pollo" does not accidentally match "repollo".
func NormalizeSearchText(value string) string {
    var builder strings.Builder
    lastSpace := true
    for _, r := range strings.ToLower(strings.TrimSpace(value)) {
        switch r {
        case 'á', 'à', 'ä', 'â':
            r = 'a'
        case 'é', 'è', 'ë', 'ê':
            r = 'e'
        case 'í', 'ì', 'ï', 'î':
            r = 'i'
        case 'ó', 'ò', 'ö', 'ô':
            r = 'o'
        case 'ú', 'ù', 'ü', 'û':
            r = 'u'
        case 'ñ':
            r = 'n'
        }
        if unicode.IsLetter(r) || unicode.IsDigit(r) {
            builder.WriteRune(r)
            lastSpace = false
            continue
        }
        if !lastSpace {
            builder.WriteByte(' ')
            lastSpace = true
        }
    }
    return strings.TrimSpace(builder.String())
}

func SearchTextMatches(value, query string) bool {
    normalizedQuery := NormalizeSearchText(query)
    if normalizedQuery == "" {
        return true
    }
    normalizedValue := NormalizeSearchText(value)
    tokens := make(map[string]struct{})
    for _, token := range strings.Fields(normalizedValue) {
        tokens[token] = struct{}{}
    }
    for _, term := range strings.Fields(normalizedQuery) {
        if len([]rune(term)) == 1 {
            if !strings.Contains(normalizedValue, term) {
                return false
            }
            continue
        }
        if _, ok := tokens[term]; !ok {
            return false
        }
    }
    return true
}

func (m *MemoryStore) Search(_ context.Context, params SearchParams) ([]Product, error) {
    query := NormalizeSearchText(params.Query)
    postalCode := strings.TrimSpace(params.PostalCode)
    scope, ok := NormalizeSearchScope(params.Scope)
    if !ok {
        return []Product{}, nil
    }

    out := make([]Product, 0)
    for _, product := range m.products {
        if query != "" && !SearchTextMatches(product.Name+" "+product.Brand, query) {
            continue
        }
        if postalCode != "" && product.PostalCode != "" && product.PostalCode != postalCode {
            continue
        }
        if !ProductMatchesSearchScope(product, scope) {
            continue
        }
        out = append(out, product)
    }
    return out, nil
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
