package api

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

func TestHealth(t *testing.T) {
    h := NewHandler(catalog.NewMemoryStore(nil))
    req := httptest.NewRequest(http.MethodGet, "/health", nil)
    rec := httptest.NewRecorder()

    h.Routes().ServeHTTP(rec, req)

    if rec.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rec.Code)
    }
}

func TestPublicReadCORS(t *testing.T) {
    h := NewHandler(catalog.NewMemoryStore(nil))
    req := httptest.NewRequest(http.MethodOptions, "/api/v1/ingredients/arroz_redondo/quote", nil)
    req.Header.Set("Origin", "https://example.test")
    rec := httptest.NewRecorder()

    h.Routes().ServeHTTP(rec, req)

    if rec.Code != http.StatusNoContent {
        t.Fatalf("expected 204, got %d", rec.Code)
    }
    if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
        t.Fatalf("expected wildcard public-read CORS, got %q", got)
    }
    if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "GET, OPTIONS" {
        t.Fatalf("unexpected allowed methods %q", got)
    }
}

func TestSearchRequiresQuery(t *testing.T) {
    h := NewHandler(catalog.NewMemoryStore(nil))
    req := httptest.NewRequest(http.MethodGet, "/api/v1/products/search", nil)
    rec := httptest.NewRecorder()

    h.Routes().ServeHTTP(rec, req)

    if rec.Code != http.StatusBadRequest {
        t.Fatalf("expected 400, got %d", rec.Code)
    }
}

func TestSearchReturnsMatchingProduct(t *testing.T) {
    h := NewHandler(catalog.NewMemoryStore(catalog.SeedProducts()))
    req := httptest.NewRequest(http.MethodGet, "/api/v1/products/search?q=arroz&postalCode=28001", nil)
    rec := httptest.NewRecorder()

    h.Routes().ServeHTTP(rec, req)

    if rec.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rec.Code)
    }

    var body struct {
        Count int `json:"count"`
    }
    if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
        t.Fatal(err)
    }
    if body.Count != 3 {
        t.Fatalf("expected 3 MVP demo products, got %d", body.Count)
    }
}

func TestSearchReturnsClassificationMetadata(t *testing.T) {
    store := catalog.NewMemoryStore([]catalog.Product{
        {
            ID:                   "dia-28809",
            SupermarketID:        "dia",
            ExternalID:           "28809",
            Name:                 "Arroz vaporizado Dia Arrozona 1 Kg",
            EAN:                  "8410830001016",
            SourceURL:            "https://www.dia.es/arroz-pastas-y-legumbres/arroz/p/28809",
            PostalCode:           "28001",
            SourceCategoryID:     "L106",
            SourceCategoryName:   "Arroz pastas y legumbres",
            SourceCategoryPath:   "arroz-pastas-y-legumbres/c/L106",
            ItemType:             "food_ingredient",
            NormalizedCategory:   "food.pantry.cereal.rice",
            RecipeCompatible:     true,
            ClassificationStatus: "classified",
            ClassificationScore:  0.98,
            ClassificationSource: "rules:v1",
        },
    })
    h := NewHandler(store)
    req := httptest.NewRequest(http.MethodGet, "/api/v1/products/search?q=vaporizado&postalCode=28001", nil)
    rec := httptest.NewRecorder()

    h.Routes().ServeHTTP(rec, req)

    if rec.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rec.Code)
    }

    var body struct {
        Items []catalog.Product `json:"items"`
    }
    if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
        t.Fatal(err)
    }
    if len(body.Items) != 1 {
        t.Fatalf("expected 1 product, got %d", len(body.Items))
    }

    got := body.Items[0]
    if got.SourceCategoryID != "L106" || got.ItemType != "food_ingredient" || !got.RecipeCompatible {
        t.Fatalf("classification metadata missing from JSON: %#v", got)
    }
    if got.NormalizedCategory != "food.pantry.cereal.rice" || got.ClassificationSource != "rules:v1" {
        t.Fatalf("unexpected classification metadata: %#v", got)
    }
    if got.EAN != "8410830001016" || got.SourceURL == "" {
        t.Fatalf("product identifiers missing from JSON: %#v", got)
    }
}
