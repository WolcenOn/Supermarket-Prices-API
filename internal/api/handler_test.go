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
