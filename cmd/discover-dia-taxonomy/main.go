package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "os"
    "time"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/supermarkets/dia"
)

func main() {
    var (
        sitemapURL = flag.String("sitemap-url", "https://www.dia.es/sitemap.xml", "DIA sitemap URL used for discovery")
        limit = flag.Int("limit", 500, "maximum number of categories returned; 0 means no result limit")
        includeNonCatalog = flag.Bool("include-non-catalog", false, "include campaign/non-catalog category IDs that do not match L<digits>")
        timeout = flag.Duration("timeout", 60*time.Second, "overall discovery timeout")
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

    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    encoder.SetEscapeHTML(false)
    if err := encoder.Encode(result); err != nil {
        fmt.Fprintf(os.Stderr, "encode result: %v\n", err)
        os.Exit(1)
    }
}
