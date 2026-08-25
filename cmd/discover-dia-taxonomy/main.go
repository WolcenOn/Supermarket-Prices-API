package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "io"
    "os"
    "strings"
    "time"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/supermarkets/dia"
)

type taxonomyNDJSONSummary struct {
    Type               string `json:"type"`
    SitemapURL         string `json:"sitemapUrl"`
    DocumentsScanned   int    `json:"documentsScanned"`
    URLEntriesScanned  int    `json:"urlEntriesScanned"`
    CandidatesFound    int    `json:"candidatesFound"`
    ExcludedNonCatalog int    `json:"excludedNonCatalog"`
    Truncated          bool   `json:"truncated"`
    CategoryCount      int    `json:"categoryCount"`
}

type taxonomyNDJSONCategory struct {
    Type       string `json:"type"`
    ID         string `json:"id"`
    Name       string `json:"name"`
    Path       string `json:"path"`
    ParentPath string `json:"parentPath,omitempty"`
    URL        string `json:"url"`
    Depth      int    `json:"depth"`
}

func main() {
    var (
        sitemapURL = flag.String("sitemap-url", "https://www.dia.es/sitemap.xml", "DIA sitemap URL used for discovery")
        limit = flag.Int("limit", 500, "maximum number of categories returned; 0 means no result limit")
        includeNonCatalog = flag.Bool("include-non-catalog", false, "include campaign/non-catalog category IDs that do not match L<digits>")
        timeout = flag.Duration("timeout", 60*time.Second, "overall discovery timeout")
        format = flag.String("format", "json", "output format: json or ndjson")
    )
    flag.Parse()

    ctx, cancel := context.WithTimeout(context.Background(), *timeout)
    defer cancel()

    discoverer := dia.NewTaxonomyDiscoverer()
    result, err := discoverer.Discover(ctx, *sitemapURL, dia.TaxonomyOptions{
        Limit: *limit,
        IncludeNonCatalog: *includeNonCatalog,
    })
    if err != nil {
        fmt.Fprintf(os.Stderr, "discover DIA taxonomy: %v\n", err)
        os.Exit(1)
    }

    if err := writeTaxonomyResult(os.Stdout, result, *format); err != nil {
        fmt.Fprintf(os.Stderr, "encode result: %v\n", err)
        os.Exit(1)
    }
}

func writeTaxonomyResult(w io.Writer, result dia.TaxonomyResult, format string) error {
    switch strings.ToLower(strings.TrimSpace(format)) {
    case "", "json":
        encoder := json.NewEncoder(w)
        encoder.SetIndent("", "  ")
        encoder.SetEscapeHTML(false)
        return encoder.Encode(result)
    case "ndjson":
        encoder := json.NewEncoder(w)
        encoder.SetEscapeHTML(false)
        summary := taxonomyNDJSONSummary{
            Type:               "summary",
            SitemapURL:         result.SitemapURL,
            DocumentsScanned:   result.DocumentsScanned,
            URLEntriesScanned:  result.URLEntriesScanned,
            CandidatesFound:    result.CandidatesFound,
            ExcludedNonCatalog: result.ExcludedNonCatalog,
            Truncated:          result.Truncated,
            CategoryCount:      len(result.Categories),
        }
        if err := encoder.Encode(summary); err != nil {
            return err
        }
        for _, category := range result.Categories {
            line := taxonomyNDJSONCategory{
                Type:       "category",
                ID:         category.ID,
                Name:       category.Name,
                Path:       category.Path,
                ParentPath: category.ParentPath,
                URL:        category.URL,
                Depth:      category.Depth,
            }
            if err := encoder.Encode(line); err != nil {
                return err
            }
        }
        return nil
    default:
        return fmt.Errorf("unsupported format %q (use json or ndjson)", format)
    }
}
