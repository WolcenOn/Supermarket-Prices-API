package main

import (
    "context"
    "strings"
    "testing"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
    "github.com/WolcenOn/Supermarket-Prices-API/internal/importer"
    "github.com/WolcenOn/Supermarket-Prices-API/internal/supermarkets/dia"
)

type testSink struct{}

func (s *testSink) SaveProducts(context.Context, []catalog.Product) error {
    return nil
}

func TestPersistentImportSinkCanDisableMatching(t *testing.T) {
    delegate := &testSink{}
    got := persistentImportSink(nil, delegate, false)
    if got != importer.Sink(delegate) {
        t.Fatalf("matching disabled: got %T, want original delegate", got)
    }
}

func TestPersistentImportSinkKeepsExistingMatchingDefault(t *testing.T) {
    delegate := &testSink{}
    got := persistentImportSink(nil, delegate, true)
    matching, ok := got.(*matchingSink)
    if !ok {
        t.Fatalf("matching enabled: got %T, want *matchingSink", got)
    }
    if matching.delegate != importer.Sink(delegate) {
        t.Fatalf("matching enabled: delegate was not preserved")
    }
}

func TestSelectCategoryURLsByParent(t *testing.T) {
    categories := []dia.TaxonomyCategory{
        {ID: "L2022", ParentPath: "verduras", URL: "https://www.dia.es/verduras/ajos-cebollas-y-puerros/c/L2022"},
        {ID: "L2023", ParentPath: "verduras", URL: "https://www.dia.es/verduras/tomates-pimientos-y-pepinos/c/L2023"},
        {ID: "L2051", ParentPath: "huevos-leche-y-mantequilla", URL: "https://www.dia.es/huevos-leche-y-mantequilla/leche/c/L2051"},
    }

    urls, err := selectCategoryURLs(categories, []string{"verduras"}, 25)
    if err != nil {
        t.Fatalf("select categories: %v", err)
    }
    if len(urls) != 2 {
        t.Fatalf("got %d URLs, want 2: %#v", len(urls), urls)
    }
    if !strings.Contains(urls[0], "/verduras/") || !strings.Contains(urls[1], "/verduras/") {
        t.Fatalf("unexpected URLs: %#v", urls)
    }
}

func TestSelectCategoryURLsSupportsMultipleParents(t *testing.T) {
    categories := []dia.TaxonomyCategory{
        {ID: "L2023", ParentPath: "verduras", URL: "https://www.dia.es/verduras/tomates-pimientos-y-pepinos/c/L2023"},
        {ID: "L2051", ParentPath: "huevos-leche-y-mantequilla", URL: "https://www.dia.es/huevos-leche-y-mantequilla/leche/c/L2051"},
        {ID: "L2108", ParentPath: "agua-y-refrescos", URL: "https://www.dia.es/agua-y-refrescos/cola/c/L2108"},
    }

    urls, err := selectCategoryURLs(categories, []string{"verduras", "huevos-leche-y-mantequilla"}, 25)
    if err != nil {
        t.Fatalf("select categories: %v", err)
    }
    if len(urls) != 2 {
        t.Fatalf("got %d URLs, want 2: %#v", len(urls), urls)
    }
}

func TestSelectCategoryURLsRejectsUnknownParent(t *testing.T) {
    categories := []dia.TaxonomyCategory{
        {ID: "L2023", ParentPath: "verduras", URL: "https://www.dia.es/verduras/tomates-pimientos-y-pepinos/c/L2023"},
    }

    _, err := selectCategoryURLs(categories, []string{"verduras", "no-existe"}, 25)
    if err == nil || !strings.Contains(err.Error(), "no-existe") {
        t.Fatalf("expected missing parent error, got %v", err)
    }
}

func TestSelectCategoryURLsEnforcesSafetyLimit(t *testing.T) {
    categories := []dia.TaxonomyCategory{
        {ID: "L2022", ParentPath: "verduras", URL: "https://www.dia.es/verduras/ajos-cebollas-y-puerros/c/L2022"},
        {ID: "L2023", ParentPath: "verduras", URL: "https://www.dia.es/verduras/tomates-pimientos-y-pepinos/c/L2023"},
    }

    _, err := selectCategoryURLs(categories, []string{"verduras"}, 1)
    if err == nil || !strings.Contains(err.Error(), "category-limit=1") {
        t.Fatalf("expected category limit error, got %v", err)
    }
}

func TestResolveCategoryURLsKeepsExplicitCategories(t *testing.T) {
    urls, err := resolveCategoryURLs(context.Background(), "https://example.test/a, https://example.test/b", "", "", 25)
    if err != nil {
        t.Fatalf("resolve explicit categories: %v", err)
    }
    if len(urls) != 2 || urls[0] != "https://example.test/a" || urls[1] != "https://example.test/b" {
        t.Fatalf("unexpected explicit URLs: %#v", urls)
    }
}
