package dia

import (
    "context"
    "fmt"
    "html"
    "io"
    "net/http"
    "regexp"
    "strings"
    "time"
)

var (
    tagRE       = regexp.MustCompile(`(?s)<[^>]*>`)
    whitespace = regexp.MustCompile(`[\t\f\v ]+`)
)

const defaultBrowserUserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36 Supermarket-Prices-API/0.1"

// HTTPSource fetches a curated set of public DIA category/listing pages.
// It deliberately does not call DIA search routes. Search is performed later
// against our own persisted catalog.
type HTTPSource struct {
    Client       *http.Client
    CategoryURLs []string
    UserAgent    string
    Now          func() time.Time
}

func NewHTTPSource(categoryURLs []string) *HTTPSource {
    return &HTTPSource{
        Client:       &http.Client{Timeout: 20 * time.Second},
        CategoryURLs: append([]string(nil), categoryURLs...),
        UserAgent:    defaultBrowserUserAgent,
        Now:          time.Now,
    }
}

func (s *HTTPSource) Search(ctx context.Context, query, postalCode string) ([]RawProduct, error) {
    if s.Client == nil {
        s.Client = &http.Client{Timeout: 20 * time.Second}
    }
    if s.Now == nil {
        s.Now = time.Now
    }
    if strings.TrimSpace(s.UserAgent) == "" {
        s.UserAgent = defaultBrowserUserAgent
    }

    var products []RawProduct
    for _, categoryURL := range s.CategoryURLs {
        categoryURL = strings.TrimSpace(categoryURL)
        if categoryURL == "" {
            continue
        }

        req, err := http.NewRequestWithContext(ctx, http.MethodGet, categoryURL, nil)
        if err != nil {
            return nil, fmt.Errorf("dia: build category request: %w", err)
        }
        req.Header.Set("User-Agent", s.UserAgent)
        req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
        req.Header.Set("Accept-Language", "es-ES,es;q=0.9,en;q=0.5")
        req.Header.Set("Cache-Control", "no-cache")
        req.Header.Set("Pragma", "no-cache")

        resp, err := s.Client.Do(req)
        if err != nil {
            return nil, fmt.Errorf("dia: fetch %s: %w", categoryURL, err)
        }

        body, readErr := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
        resp.Body.Close()
        if readErr != nil {
            return nil, fmt.Errorf("dia: read %s: %w", categoryURL, readErr)
        }
        if resp.StatusCode < 200 || resp.StatusCode >= 300 {
            return nil, fmt.Errorf("dia: fetch %s: unexpected status %d", categoryURL, resp.StatusCode)
        }

        pageProducts, err := parseCategoryHTML(string(body), postalCode, s.Now().UTC())
        if err != nil {
            return nil, fmt.Errorf("dia: parse %s: %w", categoryURL, err)
        }
        products = append(products, pageProducts...)
    }

    return deduplicateRawProducts(products), nil
}

func parseCategoryHTML(document, postalCode string, observedAt time.Time) ([]RawProduct, error) {
    semantic := HTMLToSemanticText(document)
    products := ParseRenderedSnapshot(semantic, postalCode, observedAt)
    if len(products) > 0 {
        return products, nil
    }

    // A successful HTTP response with zero parsed products is not treated as a
    // valid empty category. It usually means DIA served a shell/challenge page
    // or changed its markup, and silently accepting it would hide acquisition
    // breakage from the worker.
    visibleMarkers := strings.Contains(document, "sku_id::") || strings.Contains(semantic, "sku_id::")
    return nil, fmt.Errorf("no products parsed (sku markers visible=%t, response_bytes=%d)", visibleMarkers, len(document))
}

// HTMLToSemanticText converts a DIA catalog HTML response into a line-oriented
// text representation understood by ParseRenderedSnapshot. It intentionally
// removes executable scripts and styles; product data is expected in the
// server-rendered category markup for the browser-compatible response.
func HTMLToSemanticText(document string) string {
    document = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`).ReplaceAllString(document, "\n")
    document = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`).ReplaceAllString(document, "\n")
    document = regexp.MustCompile(`(?i)<br\s*/?>|</p>|</div>|</li>|</article>|</section>|</button>`).ReplaceAllString(document, "\n")
    document = tagRE.ReplaceAllString(document, " ")
    document = html.UnescapeString(document)

    lines := strings.Split(strings.ReplaceAll(document, "\r", "\n"), "\n")
    cleaned := make([]string, 0, len(lines))
    for _, line := range lines {
        line = whitespace.ReplaceAllString(strings.TrimSpace(line), " ")
        if line != "" {
            cleaned = append(cleaned, line)
        }
    }
    return strings.Join(cleaned, "\n")
}

func deduplicateRawProducts(products []RawProduct) []RawProduct {
    seen := make(map[string]int, len(products))
    out := make([]RawProduct, 0, len(products))
    for _, product := range products {
        key := strings.TrimSpace(product.ExternalID)
        if key == "" {
            continue
        }
        if index, ok := seen[key]; ok {
            out[index] = product
            continue
        }
        seen[key] = len(out)
        out = append(out, product)
    }
    return out
}
