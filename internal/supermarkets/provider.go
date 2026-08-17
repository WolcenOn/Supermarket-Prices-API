package supermarkets

import (
    "context"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

// Provider isolates the acquisition logic for one supermarket.
// Implementations may use an official API, a documented data feed, HTML parsing,
// or browser automation where appropriate and permitted. Consumers of the service
// should never depend on source-specific response formats.
type Provider interface {
    ID() string
    Search(ctx context.Context, query, postalCode string) ([]catalog.Product, error)
}
