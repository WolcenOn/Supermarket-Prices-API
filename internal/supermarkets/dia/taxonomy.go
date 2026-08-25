package dia

import (
    "context"
    "encoding/xml"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "regexp"
    "sort"
    "strings"
    "time"
)

var catalogCategoryIDRE = regexp.MustCompile(`^L[0-9]+$`)

const (
    defaultTaxonomyMaxDocuments = 100
    defaultTaxonomyMaxBytes     = int64(16 << 20)
    defaultTaxonomyMaxDepth     = 4
)

type TaxonomyCategory struct {
    ID         string `json:"id"`
    Name       string `json:"name"`
    Path       string `json:"path"`
    ParentPath string `json:"parentPath,omitempty"`
    URL        string `json:"url"`
    Depth      int    `json:"depth"`
}

type TaxonomyOptions struct {
    Limit             int
    IncludeNonCatalog bool
}

type TaxonomyResult struct {
    SitemapURL         string             `json:"sitemapUrl"`
    DocumentsScanned  int                `json:"documentsScanned"`
    URLEntriesScanned int                `json:"urlEntriesScanned"`
    CandidatesFound   int                `json:"candidatesFound"`
    ExcludedNonCatalog int               `json:"excludedNonCatalog"`
    Truncated          bool               `json:"truncated"`
    Categories         []TaxonomyCategory `json:"categories"`
}

type TaxonomyDiscoverer struct {
    Client       *http.Client
    UserAgent    string
    MaxDocuments int
    MaxBytes     int64
    MaxDepth     int
}

func NewTaxonomyDiscoverer() *TaxonomyDiscoverer {
    return &TaxonomyDiscoverer{
        Client:       &http.Client{Timeout: 20 * time.Second},
        UserAgent:    defaultBrowserUserAgent,
        MaxDocuments: defaultTaxonomyMaxDocuments,
        MaxBytes:     defaultTaxonomyMaxBytes,
        MaxDepth:     defaultTaxonomyMaxDepth,
    }
}

func (d *TaxonomyDiscoverer) ensureDefaults() {
    if d.Client == nil {
        d.Client = &http.Client{Timeout: 20 * time.Second}
    }
    if strings.TrimSpace(d.UserAgent) == "" {
        d.UserAgent = defaultBrowserUserAgent
    }
    if d.MaxDocuments <= 0 {
        d.MaxDocuments = defaultTaxonomyMaxDocuments
    }
    if d.MaxBytes <= 0 {
        d.MaxBytes = defaultTaxonomyMaxBytes
    }
    if d.MaxDepth <= 0 {
        d.MaxDepth = defaultTaxonomyMaxDepth
    }
}

func (d *TaxonomyDiscoverer) Discover(ctx context.Context, sitemapURL string, options TaxonomyOptions) (TaxonomyResult, error) {
    d.ensureDefaults()

    root, err := url.Parse(strings.TrimSpace(sitemapURL))
    if err != nil || root.Scheme == "" || root.Hostname() == "" {
        return TaxonomyResult{}, fmt.Errorf("dia taxonomy: invalid sitemap URL")
    }
    if root.Scheme != "https" && root.Scheme != "http" {
        return TaxonomyResult{}, fmt.Errorf("dia taxonomy: unsupported sitemap scheme %q", root.Scheme)
    }

    result := TaxonomyResult{SitemapURL: root.String()}
    seenDocuments := make(map[string]struct{})
    categories := make(map[string]TaxonomyCategory)

    var walk func(string, int) error
    walk = func(documentURL string, depth int) error {
        if depth > d.MaxDepth {
            result.Truncated = true
            return nil
        }
        if result.DocumentsScanned >= d.MaxDocuments {
            result.Truncated = true
            return nil
        }
        if _, ok := seenDocuments[documentURL]; ok {
            return nil
        }
        parsed, err := url.Parse(documentURL)
        if err != nil || !sameTaxonomyHost(root, parsed) {
            return nil
        }
        seenDocuments[documentURL] = struct{}{}

        body, err := d.fetchXML(ctx, documentURL)
        if err != nil {
            return err
        }
        result.DocumentsScanned++

        index, set, err := parseSitemapDocument(body)
        if err != nil {
            return fmt.Errorf("dia taxonomy: parse %s: %w", documentURL, err)
        }

        if len(index) > 0 {
            for _, child := range index {
                if err := walk(child, depth+1); err != nil {
                    return err
                }
                if result.Truncated && result.DocumentsScanned >= d.MaxDocuments {
                    break
                }
            }
            return nil
        }

        result.URLEntriesScanned += len(set)
        for _, candidateURL := range set {
            category, ok := taxonomyCategoryFromURL(candidateURL)
            if !ok {
                continue
            }
            result.CandidatesFound++
            if !options.IncludeNonCatalog && !catalogCategoryIDRE.MatchString(category.ID) {
                result.ExcludedNonCatalog++
                continue
            }
            if existing, exists := categories[category.ID]; !exists || preferTaxonomyCategory(category, existing) {
                categories[category.ID] = category
            }
        }
        return nil
    }

    if err := walk(root.String(), 0); err != nil {
        return TaxonomyResult{}, err
    }

    result.Categories = make([]TaxonomyCategory, 0, len(categories))
    for _, category := range categories {
        result.Categories = append(result.Categories, category)
    }
    sort.Slice(result.Categories, func(i, j int) bool {
        if result.Categories[i].Path == result.Categories[j].Path {
            return result.Categories[i].ID < result.Categories[j].ID
        }
        return result.Categories[i].Path < result.Categories[j].Path
    })

    if options.Limit > 0 && len(result.Categories) > options.Limit {
        result.Categories = result.Categories[:options.Limit]
        result.Truncated = true
    }
    return result, nil
}

func (d *TaxonomyDiscoverer) fetchXML(ctx context.Context, documentURL string) ([]byte, error) {
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, documentURL, nil)
    if err != nil {
        return nil, fmt.Errorf("dia taxonomy: build request: %w", err)
    }
    req.Header.Set("User-Agent", d.UserAgent)
    req.Header.Set("Accept", "application/xml,text/xml;q=0.9,*/*;q=0.5")
    req.Header.Set("Accept-Language", "es-ES,es;q=0.9")

    resp, err := d.Client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("dia taxonomy: fetch %s: %w", documentURL, err)
    }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return nil, fmt.Errorf("dia taxonomy: fetch %s: unexpected status %d", documentURL, resp.StatusCode)
    }

    body, err := io.ReadAll(io.LimitReader(resp.Body, d.MaxBytes+1))
    if err != nil {
        return nil, fmt.Errorf("dia taxonomy: read %s: %w", documentURL, err)
    }
    if int64(len(body)) > d.MaxBytes {
        return nil, fmt.Errorf("dia taxonomy: sitemap document exceeds %d bytes", d.MaxBytes)
    }
    return body, nil
}

type sitemapIndexXML struct {
    Locations []string `xml:"sitemap>loc"`
}

type sitemapURLSetXML struct {
    Locations []string `xml:"url>loc"`
}

func parseSitemapDocument(body []byte) (index []string, urls []string, err error) {
    var root struct {
        XMLName xml.Name
    }
    if err := xml.Unmarshal(body, &root); err != nil {
        return nil, nil, err
    }

    switch strings.ToLower(root.XMLName.Local) {
    case "sitemapindex":
        var parsed sitemapIndexXML
        if err := xml.Unmarshal(body, &parsed); err != nil {
            return nil, nil, err
        }
        return cleanLocations(parsed.Locations), nil, nil
    case "urlset":
        var parsed sitemapURLSetXML
        if err := xml.Unmarshal(body, &parsed); err != nil {
            return nil, nil, err
        }
        return nil, cleanLocations(parsed.Locations), nil
    default:
        return nil, nil, fmt.Errorf("unsupported root element %q", root.XMLName.Local)
    }
}

func cleanLocations(values []string) []string {
    out := make([]string, 0, len(values))
    for _, value := range values {
        value = strings.TrimSpace(value)
        if value != "" {
            out = append(out, value)
        }
    }
    return out
}

func sameTaxonomyHost(root, candidate *url.URL) bool {
    if root == nil || candidate == nil {
        return false
    }
    return strings.EqualFold(root.Hostname(), candidate.Hostname()) && candidate.Scheme == root.Scheme
}

func taxonomyCategoryFromURL(rawURL string) (TaxonomyCategory, bool) {
    parsed, err := url.Parse(strings.TrimSpace(rawURL))
    if err != nil || parsed.Hostname() == "" {
        return TaxonomyCategory{}, false
    }

    id, name, path := sourceCategoryFromURL(rawURL)
    if id == "" || !strings.Contains("/"+path+"/", "/c/") {
        return TaxonomyCategory{}, false
    }

    segments := strings.Split(path, "/")
    categoryIndex := -1
    for i, segment := range segments {
        if segment == "c" && i+1 < len(segments) {
            categoryIndex = i
            break
        }
    }
    if categoryIndex < 1 {
        return TaxonomyCategory{}, false
    }

    parentPath := strings.Join(segments[:categoryIndex-1], "/")
    return TaxonomyCategory{
        ID:         id,
        Name:       name,
        Path:       path,
        ParentPath: parentPath,
        URL:        parsed.String(),
        Depth:      categoryIndex,
    }, true
}

func preferTaxonomyCategory(candidate, existing TaxonomyCategory) bool {
    if candidate.Depth != existing.Depth {
        return candidate.Depth < existing.Depth
    }
    if len(candidate.Path) != len(existing.Path) {
        return len(candidate.Path) < len(existing.Path)
    }
    return candidate.Path < existing.Path
}
