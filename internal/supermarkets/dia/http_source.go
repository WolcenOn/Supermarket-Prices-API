package dia

import (
    "context"
    "fmt"
    "html"
    "io"
    "net/http"
    "net/url"
    "regexp"
    "strings"
    "time"
    "unicode"
)

var (
    tagRE         = regexp.MustCompile(`(?s)<[^>]*>`)
    whitespace    = regexp.MustCompile(`[\t\f\v ]+`)
    productHrefRE = regexp.MustCompile(`(?is)href\s*=\s*["']([^"']*/p/([0-9]+)(?:[?#][^"']*)?)["']`)
)

const defaultBrowserUserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36 Supermarket-Prices-API/0.1"

// HTTPStatusError identifies a non-success response from a DIA catalog page.
// Callers that need best-effort auditing can inspect the status code without
// relying on error-string parsing; regular import callers still receive an error.
type HTTPStatusError struct {
    URL        string
    StatusCode int
}

func (e *HTTPStatusError) Error() string {
    return fmt.Sprintf("dia: fetch %s: unexpected status %d", e.URL, e.StatusCode)
}

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

func (s *HTTPSource) ensureDefaults() {
    if s.Client == nil {
        s.Client = &http.Client{Timeout: 20 * time.Second}
    }
    if s.Now == nil {
        s.Now = time.Now
    }
    if strings.TrimSpace(s.UserAgent) == "" {
        s.UserAgent = defaultBrowserUserAgent
    }
}

func (s *HTTPSource) setBrowserHeaders(req *http.Request) {
    req.Header.Set("User-Agent", s.UserAgent)
    req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
    req.Header.Set("Accept-Language", "es-ES,es;q=0.9,en;q=0.5")
    req.Header.Set("Cache-Control", "no-cache")
    req.Header.Set("Pragma", "no-cache")
}

func (s *HTTPSource) Search(ctx context.Context, query, postalCode string) ([]RawProduct, error) {
    s.ensureDefaults()

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
        s.setBrowserHeaders(req)

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
            return nil, &HTTPStatusError{URL: categoryURL, StatusCode: resp.StatusCode}
        }

        document := string(body)
        pageProducts, err := parseCategoryHTML(document, postalCode, s.Now().UTC())
        if err != nil {
            return nil, fmt.Errorf("dia: parse %s: %w", categoryURL, err)
        }
        attachProductSourceURLs(document, categoryURL, pageProducts)
        categoryID, categoryName, categoryPath := sourceCategoryFromURL(categoryURL)
        for i := range pageProducts {
            pageProducts[i].SourceCategoryID = categoryID
            pageProducts[i].SourceCategoryName = categoryName
            pageProducts[i].SourceCategoryPath = categoryPath
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

    visibleMarkers := strings.Contains(document, "sku_id") || strings.Contains(semantic, "sku_id")
    snippet := firstSKUSnippet(semantic)
    return nil, fmt.Errorf("no products parsed (sku markers visible=%t, response_bytes=%d, first_sku=%q)", visibleMarkers, len(document), snippet)
}

func attachProductSourceURLs(document, pageURL string, products []RawProduct) {
    base, err := url.Parse(strings.TrimSpace(pageURL))
    if err != nil {
        return
    }

    bySKU := make(map[string]string)
    for _, match := range productHrefRE.FindAllStringSubmatch(document, -1) {
        if len(match) < 3 {
            continue
        }
        href := html.UnescapeString(strings.TrimSpace(match[1]))
        sku := strings.TrimSpace(match[2])
        ref, err := url.Parse(href)
        if err != nil {
            continue
        }
        resolved := base.ResolveReference(ref)
        if !isPublicDIAProductURL(resolved) {
            continue
        }
        resolved.RawQuery = ""
        resolved.Fragment = ""
        if _, exists := bySKU[sku]; !exists {
            bySKU[sku] = resolved.String()
        }
    }

    for i := range products {
        if sourceURL, ok := bySKU[strings.TrimSpace(products[i].ExternalID)]; ok {
            products[i].SourceURL = sourceURL
        }
    }
}

func isPublicDIAProductURL(parsed *url.URL) bool {
    if parsed == nil || parsed.Scheme != "https" {
        return false
    }
    host := strings.ToLower(parsed.Hostname())
    if host != "dia.es" && host != "www.dia.es" {
        return false
    }
    return regexp.MustCompile(`/p/[0-9]+/?$`).MatchString(parsed.Path)
}

// HTMLToSemanticText converts DIA category HTML into a line-oriented snapshot.
// Product names/prices live in anchors and spans, so closing those elements is
// treated as a semantic boundary rather than flattening the entire card into a
// single line.
func HTMLToSemanticText(document string) string {
    document = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`).ReplaceAllString(document, "\n")
    document = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`).ReplaceAllString(document, "\n")
    document = regexp.MustCompile(`(?i)<br\s*/?>|</p>|</div>|</li>|</article>|</section>|</button>|</a>|</span>|</strong>|</h1>|</h2>|</h3>`).ReplaceAllString(document, "\n")
    document = tagRE.ReplaceAllString(document, " ")
    document = html.UnescapeString(document)

    // DIA sometimes emits the product marker inside a larger text node. Force
    // it onto its own line while preserving any content that follows it.
    markerRE := regexp.MustCompile(`sku_id\s*::\s*([0-9]+)`)
    document = markerRE.ReplaceAllString(document, "\nsku_id::$1\n")

    lines := strings.Split(strings.ReplaceAll(document, "\r", "\n"), "\n")
    cleaned := make([]string, 0, len(lines))
    for _, line := range lines {
        line = normalizeVisibleWhitespace(line)
        line = whitespace.ReplaceAllString(line, " ")
        if line != "" {
            cleaned = append(cleaned, line)
        }
    }
    return strings.Join(cleaned, "\n")
}

func sourceCategoryFromURL(rawURL string) (id, name, path string) {
    parsed, err := url.Parse(strings.TrimSpace(rawURL))
    if err != nil {
        return "", "", ""
    }

    path = strings.Trim(parsed.Path, "/")
    segments := strings.Split(path, "/")
    categorySlug := ""
    for i, segment := range segments {
        if segment == "c" && i+1 < len(segments) {
            id = strings.TrimSpace(segments[i+1])
            if i > 0 {
                categorySlug = strings.TrimSpace(segments[i-1])
            }
            break
        }
    }

    if categorySlug == "" && len(segments) > 0 {
        categorySlug = strings.TrimSpace(segments[0])
    }
    name = humanizeCategorySlug(categorySlug)
    return id, name, path
}

func humanizeCategorySlug(slug string) string {
    value := strings.TrimSpace(strings.ReplaceAll(slug, "-", " "))
    if value == "" {
        return ""
    }
    runes := []rune(value)
    runes[0] = unicode.ToUpper(runes[0])
    return string(runes)
}

func firstSKUSnippet(semantic string) string {
    lower := strings.ToLower(semantic)
    idx := strings.Index(lower, "sku_id")
    if idx < 0 {
        return ""
    }
    end := idx + 320
    if end > len(semantic) {
        end = len(semantic)
    }
    snippet := semantic[idx:end]
    snippet = strings.ReplaceAll(snippet, "\n", " | ")
    return snippet
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
