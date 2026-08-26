package api

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

type quoteAvailabilityTestStore struct {
    items []catalog.IngredientProduct
}

func (s *quoteAvailabilityTestStore) Supermarkets(context.Context) ([]catalog.Supermarket, error) {
    return nil, nil
}

func (s *quoteAvailabilityTestStore) Search(context.Context, catalog.SearchParams) ([]catalog.Product, error) {
    return nil, nil
}

func (s *quoteAvailabilityTestStore) Ingredients(context.Context) ([]catalog.CanonicalIngredient, error) {
    return nil, nil
}

func (s *quoteAvailabilityTestStore) SearchIngredients(context.Context, string) ([]catalog.CanonicalIngredient, error) {
    return nil, nil
}

func (s *quoteAvailabilityTestStore) IngredientProducts(context.Context, string, string) ([]catalog.IngredientProduct, error) {
    return s.items, nil
}

func TestIngredientQuoteSkipsUnavailableProducts(t *testing.T) {
    store := &quoteAvailabilityTestStore{
        items: []catalog.IngredientProduct{
            {
                Product: catalog.Product{
                    ID:            "unavailable-cheap",
                    SupermarketID: "dia",
                    Name:          "Tomate canario malla 750 g",
                    PackageAmount: 750,
                    PackageUnit:   "g",
                    Price:         1.25,
                    Available:     false,
                },
            },
            {
                Product: catalog.Product{
                    ID:            "available",
                    SupermarketID: "dia",
                    Name:          "Tomate pera bandeja 500 g",
                    PackageAmount: 500,
                    PackageUnit:   "g",
                    Price:         1.39,
                    Available:     true,
                },
            },
        },
    }

    h := NewHandler(store)
    req := httptest.NewRequest(http.MethodGet, "/api/v1/ingredients/tomate/quote?amount=600&unit=g&postalCode=28001", nil)
    rec := httptest.NewRecorder()

    h.Routes().ServeHTTP(rec, req)

    if rec.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
    }

    var body struct {
        Count   int                     `json:"count"`
        Skipped int                     `json:"skipped"`
        Items   []catalog.PurchaseQuote `json:"items"`
    }
    if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
        t.Fatal(err)
    }
    if body.Count != 1 || len(body.Items) != 1 {
        t.Fatalf("expected only the available quote, got %#v", body)
    }
    if body.Skipped != 1 {
        t.Fatalf("expected unavailable product to be skipped, got skipped=%d", body.Skipped)
    }
    if body.Items[0].Product.ID != "available" {
        t.Fatalf("unexpected quoted product: %#v", body.Items[0].Product)
    }
    if body.Items[0].TotalCost != 2.78 {
        t.Fatalf("expected checkout cost 2.78, got %.2f", body.Items[0].TotalCost)
    }
}
