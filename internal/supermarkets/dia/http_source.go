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
        UserAgent:    "Supermarket-Prices-API/0.1 (+https://github.com/WolcenOn/Supermarket-Prices-API)",
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
        req.Header.Set("Accept", "text/html,application/xhtml+xml")
        req.Header.Set("Accept-Language", "es-ES,es;q=0.9")

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

        semantic := HTMLToSemanticText(string(body))
        pageProducts := ParseRenderedSnapshot(semantic, postalCode, s.Now().UTC())
        products = append(products, pageProducts...)
    }

    return deduplicateRawProducts(products), nil
}

// HTMLToSemanticText converts a DIA catalog HTML response into a line-oriented
// text representation understood by ParseRenderedSnapshot. It is intentionally
// conservative: acquisition-specific cleanup lives here, while product
// normalization remains independent in Normalize.
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
