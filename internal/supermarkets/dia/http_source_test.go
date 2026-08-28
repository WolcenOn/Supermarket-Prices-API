package dia

import (
    "context"
    "errors"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
    "time"
)

func TestParseCategoryHTMLCurrentServerRenderedShape(t *testing.T) {
    document := `<!doctype html><html><body>
<div>condition :: true index::0 initialLoadedItems::10 item.isVisible:: item.type:: item.sku_id::5873 item.data::<a>Arroz largo Dia Arrozona 1 Kg</a><span>1,20&nbsp;€</span><span>(1,20&nbsp;€/KILO)</span><button>Añadir</button></div>
<div>condition :: true index::1 initialLoadedItems::10 item.isVisible:: item.type:: item.sku_id :: 151 item.data::<a>Arroz extra Dia Arrozona 1 Kg</a><span>1,20&nbsp;€</span><span>(1,20&nbsp;€/KILO)</span><button>Añadir</button></div>
</body></html>`

    products, err := parseCategoryHTML(document, "28001", time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC))
    if err != nil {
        t.Fatal(err)
    }
    if len(products) != 2 {
        t.Fatalf("expected 2 products, got %d", len(products))
    }
    if products[0].ExternalID != "5873" || products[0].Name != "Arroz largo Dia Arrozona 1 Kg" || products[0].RegularPrice != 1.20 {
        t.Fatalf("unexpected first product: %+v", products[0])
    }
    if products[1].ExternalID != "151" || products[1].RegularPrice != 1.20 {
        t.Fatalf("unexpected second product: %+v", products[1])
    }
}

func TestParseCategoryHTMLRejectsSuccessfulShellPage(t *testing.T) {
    _, err := parseCategoryHTML(`<html><body><div id="app">Productos</div></body></html>`, "28001", time.Time{})
    if err == nil {
        t.Fatal("expected empty shell to fail")
    }
    if !strings.Contains(err.Error(), "no products parsed") || !strings.Contains(err.Error(), "sku markers visible=false") {
        t.Fatalf("unexpected error: %v", err)
    }
}

func TestNewHTTPSourceUsesBrowserCompatibleUserAgent(t *testing.T) {
    source := NewHTTPSource(nil)
    if !strings.Contains(source.UserAgent, "Mozilla/5.0") || !strings.Contains(source.UserAgent, "Supermarket-Prices-API") {
        t.Fatalf("unexpected user agent: %q", source.UserAgent)
    }
}

func TestHTTPSourceReturnsTypedStatusError(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
        http.NotFound(w, nil)
    }))
    defer server.Close()

    source := NewHTTPSource([]string{server.URL + "/category"})
    _, err := source.Search(context.Background(), "catalog-scan", "28001")
    if err == nil {
        t.Fatal("expected HTTP status error")
    }

    var statusErr *HTTPStatusError
    if !errors.As(err, &statusErr) {
        t.Fatalf("expected HTTPStatusError, got %T: %v", err, err)
    }
    if statusErr.StatusCode != http.StatusNotFound {
        t.Fatalf("status = %d, want %d", statusErr.StatusCode, http.StatusNotFound)
    }
    if statusErr.URL != server.URL+"/category" {
        t.Fatalf("url = %q, want %q", statusErr.URL, server.URL+"/category")
    }
}
