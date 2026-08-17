package importer

import (
    "context"
    "fmt"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

// Provider is the minimal contract needed by the importer.
type Provider interface {
    ID() string
    Search(ctx context.Context, query, postalCode string) ([]catalog.Product, error)
}

// Sink persists normalized products and their price observations.
// PostgreSQL will implement this interface; tests can use an in-memory sink.
type Sink interface {
    SaveProducts(ctx context.Context, products []catalog.Product) error
}

type Result struct {
    Supermarket string
    Found       int
    Saved       int
}

func Run(ctx context.Context, provider Provider, sink Sink, query, postalCode string) (Result, error) {
    products, err := provider.Search(ctx, query, postalCode)
    if err != nil {
        return Result{}, fmt.Errorf("import %s: acquire products: %w", provider.ID(), err)
    }

    result := Result{Supermarket: provider.ID(), Found: len(products)}
    if sink == nil {
        return result, nil
    }
    if err := sink.SaveProducts(ctx, products); err != nil {
        return result, fmt.Errorf("import %s: persist products: %w", provider.ID(), err)
    }
    result.Saved = len(products)
    return result, nil
}
