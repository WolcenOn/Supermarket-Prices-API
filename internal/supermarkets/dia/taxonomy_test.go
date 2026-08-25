package dia

import (
    "context"
    "fmt"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestParseSitemapDocumentURLSet(t *testing.T) {
    _, urls, err := parseSitemapDocument([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>https://www.dia.es/carnes/c/L102</loc></url>
  <url><loc>https://www.dia.es/huevos-leche-y-mantequilla/leche/c/L2051</loc></url>
</urlset>`))
    if err != nil {
        t.Fatal(err)
    }
    if len(urls) != 2 {
        t.Fatalf("expected 2 URLs, got %d", len(urls))
    }
}

func TestTaxonomyCategoryFromURL(t *testing.T) {
    category, ok := taxonomyCategoryFromURL("https://www.dia.es/huevos-leche-y-mantequilla/leche/c/L2051")
    if !ok {
        t.Fatal("expected category URL")
    }
    if category.ID != "L2051" {
        t.Fatalf("unexpected id %q", category.ID)
    }
    if category.Name != "Leche" {
        t.Fatalf("unexpected name %q", category.Name)
    }
    if category.ParentPath != "huevos-leche-y-mantequilla" {
        t.Fatalf("unexpected parent path %q", category.ParentPath)
    }
    if category.Depth != 2 {
        t.Fatalf("unexpected depth %d", category.Depth)
    }
}

func TestDiscoverTaxonomyRecursesAndFiltersCatalogIDs(t *testing.T) {
    mux := http.NewServeMux()
    server := httptest.NewServer(mux)
    defer server.Close()

    mux.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, `<?xml version="1.0"?><sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
<sitemap><loc>%s/catalog.xml</loc></sitemap>
</sitemapindex>`, server.URL)
    })
    mux.HandleFunc("/catalog.xml", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, `<?xml version="1.0"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
<url><loc>%s/carnes/c/L102</loc></url>
<url><loc>%s/huevos-leche-y-mantequilla/leche/c/L2051</loc></url>
<url><loc>%s/novedades-y-recomendados/especial-mundial/dia/c/13lgF136o</loc></url>
<url><loc>%s/huevos-leche-y-mantequilla/leche/c/L2051</loc></url>
<url><loc>%s/producto/p/607</loc></url>
</urlset>`, server.URL, server.URL, server.URL, server.URL, server.URL)
    })

    discoverer := NewTaxonomyDiscoverer()
    discoverer.Client = server.Client()
    result, err := discoverer.Discover(context.Background(), server.URL+"/sitemap.xml", TaxonomyOptions{})
    if err != nil {
        t.Fatal(err)
    }

    if result.DocumentsScanned != 2 {
        t.Fatalf("expected 2 documents, got %d", result.DocumentsScanned)
    }
    if result.URLEntriesScanned != 5 {
        t.Fatalf("expected 5 URL entries, got %d", result.URLEntriesScanned)
    }
    if result.CandidatesFound != 4 {
        t.Fatalf("expected 4 category candidates, got %d", result.CandidatesFound)
    }
    if result.ExcludedNonCatalog != 1 {
        t.Fatalf("expected 1 non-catalog exclusion, got %d", result.ExcludedNonCatalog)
    }
    if len(result.Categories) != 2 {
        t.Fatalf("expected 2 deduplicated catalog categories, got %d", len(result.Categories))
    }
    if result.Categories[0].ID != "L102" || result.Categories[1].ID != "L2051" {
        t.Fatalf("unexpected categories %#v", result.Categories)
    }
}

func TestDiscoverTaxonomyCanIncludeNonCatalogIDs(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, `<?xml version="1.0"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
<url><loc>%s/novedades-y-recomendados/especial-mundial/dia/c/13lgF136o</loc></url>
</urlset>`, serverURL(r))
    }))
    defer server.Close()

    discoverer := NewTaxonomyDiscoverer()
    discoverer.Client = server.Client()
    result, err := discoverer.Discover(context.Background(), server.URL, TaxonomyOptions{IncludeNonCatalog: true})
    if err != nil {
        t.Fatal(err)
    }
    if len(result.Categories) != 1 || result.Categories[0].ID != "13lgF136o" {
        t.Fatalf("unexpected categories %#v", result.Categories)
    }
}

func serverURL(r *http.Request) string {
    return "http://" + r.Host
}
