package api

import (
    "context"
    "errors"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

type failingStore struct{}

func (f failingStore) Supermarkets(context.Context) ([]catalog.Supermarket, error) {
    return nil, errors.New("database unavailable")
}

func (f failingStore) Search(context.Context, catalog.SearchParams) ([]catalog.Product, error) {
    return nil, errors.New("database unavailable")
}

func TestSearchReturns500WhenCatalogFails(t *testing.T) {
    h := NewHandler(failingStore{})
    req := httptest.NewRequest(http.MethodGet, "/api/v1/products/search?q=arroz", nil)
    rec := httptest.NewRecorder()

    h.Routes().ServeHTTP(rec, req)

    if rec.Code != http.StatusInternalServerError {
        t.Fatalf("expected 500, got %d", rec.Code)
    }
}

func TestSupermarketsReturns500WhenCatalogFails(t *testing.T) {
    h := NewHandler(failingStore{})
    req := httptest.NewRequest(http.MethodGet, "/api/v1/supermarkets", nil)
    rec := httptest.NewRecorder()

    h.Routes().ServeHTTP(rec, req)

    if rec.Code != http.StatusInternalServerError {
        t.Fatalf("expected 500, got %d", rec.Code)
    }
}
