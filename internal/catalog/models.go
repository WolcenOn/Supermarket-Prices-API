package catalog

import (
    "context"
    "time"
)

type Supermarket struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

type Promotion struct {
    Type        string  `json:"type"`
    Label       string  `json:"label,omitempty"`
    Price       float64 `json:"price,omitempty"`
    DiscountPct float64 `json:"discountPct,omitempty"`
}

type Product struct {
    ID             string      `json:"id"`
    SupermarketID  string      `json:"supermarketId"`
    ExternalID     string      `json:"externalId"`
    Name           string      `json:"name"`
    Brand          string      `json:"brand,omitempty"`
    PackageAmount  float64     `json:"packageAmount,omitempty"`
    PackageUnit    string      `json:"packageUnit,omitempty"`
    Price          float64     `json:"price"`
    PricePerUnit   float64     `json:"pricePerUnit,omitempty"`
    PriceUnit      string      `json:"priceUnit,omitempty"`
    PostalCode     string      `json:"postalCode,omitempty"`
    VariableWeight bool        `json:"variableWeight"`
    Available      bool        `json:"available"`
    Promotions     []Promotion `json:"promotions,omitempty"`
    ObservedAt     time.Time   `json:"observedAt"`
}

type SearchParams struct {
    Query      string
    PostalCode string
}

type Store interface {
    Supermarkets(ctx context.Context) ([]Supermarket, error)
    Search(ctx context.Context, params SearchParams) ([]Product, error)
}
