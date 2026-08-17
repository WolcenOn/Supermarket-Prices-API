package dia

import (
    "context"
    "fmt"
    "strings"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

// Source is the DIA-specific acquisition boundary.
// Implementations can use a documented/public structured source, HTML parsing,
// or a local fixture. The Provider itself only understands RawProduct values.
type Source interface {
    Search(ctx context.Context, query, postalCode string) ([]RawProduct, error)
}

type Provider struct {
    source Source
}

func NewProvider(source Source) *Provider {
    return &Provider{source: source}
}

func (p *Provider) ID() string { return supermarketID }

func (p *Provider) Search(ctx context.Context, query, postalCode string) ([]catalog.Product, error) {
    if p == nil || p.source == nil {
        return nil, fmt.Errorf("dia: source is required")
    }

    query = strings.TrimSpace(query)
    postalCode = strings.TrimSpace(postalCode)
    if query == "" {
        return nil, fmt.Errorf("dia: query is required")
    }

    rawProducts, err := p.source.Search(ctx, query, postalCode)
    if err != nil {
        return nil, fmt.Errorf("dia: search source: %w", err)
    }

    products := make([]catalog.Product, 0, len(rawProducts))
    for _, raw := range rawProducts {
        if raw.PostalCode == "" {
            raw.PostalCode = postalCode
        }
        product, err := Normalize(raw)
        if err != nil {
            continue
        }
        products = append(products, product)
    }

    return products, nil
}
