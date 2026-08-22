package api

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

type nutritionTestStore struct {
    *catalog.MemoryStore
    nutrition map[string][]catalog.ProductNutrition
}

func (s *nutritionTestStore) ProductNutrition(_ context.Context, productID string) ([]catalog.ProductNutrition, error) {
    return s.nutrition[productID], nil
}

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

func TestProductNutritionReturnsPersistedSourceWithoutInventingMissingNutrients(t *testing.T) {
    productID := "55efbb57-0b2f-4e47-bf54-58b11c27bf5c"
    energyKcal := 86.0
    proteinG := 3.8
    store := &nutritionTestStore{
        MemoryStore: catalog.NewMemoryStore(nil),
        nutrition: map[string][]catalog.ProductNutrition{
            productID: {
                {
                    ProductID:              productID,
                    SupermarketID:          "dia",
                    ExternalID:             "303311",
                    Source:                 "dia_product_page",
                    SourceURL:              "https://www.dia.es/congelados-y-helados/arroces-y-pasta/p/303311",
                    ObservedAt:             time.Date(2026, 8, 22, 19, 4, 13, 0, time.UTC),
                    DescriptionText:        "ARROZ CON MARISCO, SALSA CON TOMATE Y CALDO DE PESCADO. PRODUCTO ULTRACONGELADO.",
                    SourceIngredientsBlock: "Tipo de producto: Arroz",
                    EnergyKcal:             &energyKcal,
                    ProteinG:               &proteinG,
                },
            },
        },
    }

    h := NewHandler(store)
    req := httptest.NewRequest(http.MethodGet, "/api/v1/products/"+productID+"/nutrition", nil)
    rec := httptest.NewRecorder()
    h.Routes().ServeHTTP(rec, req)

    if rec.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
    }

    var body struct {
        ProductID string                     `json:"productId"`
        Count     int                        `json:"count"`
        Items     []catalog.ProductNutrition `json:"items"`
    }
    if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
        t.Fatal(err)
    }
    if body.ProductID != productID || body.Count != 1 || len(body.Items) != 1 {
        t.Fatalf("unexpected response: %#v", body)
    }
    got := body.Items[0]
    if got.Source != "dia_product_page" || got.EnergyKcal == nil || *got.EnergyKcal != 86 {
        t.Fatalf("unexpected nutrition: %#v", got)
    }
    if got.FiberG != nil || got.IngredientsText != "" {
        t.Fatalf("missing DIA values must remain absent: %#v", got)
    }
}
