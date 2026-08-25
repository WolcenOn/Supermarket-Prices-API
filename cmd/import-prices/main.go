package main

import (
    "context"
    "database/sql"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "os"
    "sort"
    "strings"
    "time"

    _ "github.com/lib/pq"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
    "github.com/WolcenOn/Supermarket-Prices-API/internal/importer"
    "github.com/WolcenOn/Supermarket-Prices-API/internal/matching"
    postgresstore "github.com/WolcenOn/Supermarket-Prices-API/internal/storage/postgres"
    "github.com/WolcenOn/Supermarket-Prices-API/internal/supermarkets/dia"
)

var defaultDIACategories = []string{
    // Keep the default validation set deliberately small. This public category
    // currently exposes rice SKUs, names, prices and unit prices without using
    // DIA's search routes. Category identifiers can change, so callers may
    // override this list with --categories without changing the provider.
    "https://www.dia.es/arroz-pastas-y-legumbres/c/L106",
}

type collectingSink struct {
    products []catalog.Product
}

func (s *collectingSink) SaveProducts(_ context.Context, products []catalog.Product) error {
    s.products = append(s.products, products...)
    return nil
}

type matchingSink struct {
    db       *sql.DB
    delegate importer.Sink
}

func (s *matchingSink) SaveProducts(ctx context.Context, products []catalog.Product) error {
    if err := s.delegate.SaveProducts(ctx, products); err != nil {
        return err
    }

    matches := make([]matching.Match, 0)
    for _, product := range products {
        matches = append(matches, matching.Suggest(product)...)
    }
    return postgresstore.SaveIngredientMatches(ctx, s.db, matches)
}

func persistentImportSink(db *sql.DB, delegate importer.Sink, matchProducts bool) importer.Sink {
    if !matchProducts {
        return delegate
    }
    return &matchingSink{db: db, delegate: delegate}
}

func main() {
    supermarket := flag.String("supermarket", "dia", "supermarket provider to import")
    postalCode := flag.String("postal-code", "", "postal code used for location-sensitive observations")
    categories := flag.String("categories", "", "comma-separated DIA category URLs; defaults to a small validation set")
    categoryParents := flag.String("category-parents", "", "comma-separated DIA taxonomy parent paths to discover and import, e.g. verduras,huevos-leche-y-mantequilla")
    sitemapURL := flag.String("sitemap-url", "https://www.dia.es/sitemap.xml", "DIA sitemap URL used when --category-parents is set")
    categoryLimit := flag.Int("category-limit", 25, "maximum number of discovered categories allowed for one import; 0 disables the safety limit")
    dryRun := flag.Bool("dry-run", true, "fetch and normalize without persisting")
    matchProducts := flag.Bool("match-products", true, "when persisting, also save deterministic product-to-canonical matches")
    timeout := flag.Duration("timeout", 45*time.Second, "maximum importer execution time, including taxonomy discovery")
    flag.Parse()

    if strings.ToLower(strings.TrimSpace(*supermarket)) != "dia" {
        log.Fatalf("unsupported supermarket %q; current phase supports dia", *supermarket)
    }
    if strings.TrimSpace(*categories) != "" && strings.TrimSpace(*categoryParents) != "" {
        log.Fatal("use either --categories or --category-parents, not both")
    }
    if *categoryLimit < 0 {
        log.Fatal("--category-limit must be >= 0")
    }

    ctx, cancel := context.WithTimeout(context.Background(), *timeout)
    defer cancel()

    categoryURLs, err := resolveCategoryURLs(ctx, strings.TrimSpace(*categories), strings.TrimSpace(*categoryParents), strings.TrimSpace(*sitemapURL), *categoryLimit)
    if err != nil {
        log.Fatal(err)
    }
    if strings.TrimSpace(*categoryParents) != "" {
        log.Printf("resolved %d DIA categories from parent paths %q", len(categoryURLs), *categoryParents)
    }

    source := dia.NewHTTPSource(categoryURLs)
    provider := dia.NewProvider(source)

    if *dryRun {
        runDry(ctx, provider, strings.TrimSpace(*postalCode))
        return
    }

    runPersistent(ctx, provider, strings.TrimSpace(*postalCode), *matchProducts)
}

func resolveCategoryURLs(ctx context.Context, explicitCategories, categoryParents, sitemapURL string, categoryLimit int) ([]string, error) {
    if explicitCategories != "" {
        return splitCSV(explicitCategories), nil
    }
    if categoryParents == "" {
        return append([]string(nil), defaultDIACategories...), nil
    }

    discoverer := dia.NewTaxonomyDiscoverer()
    result, err := discoverer.Discover(ctx, sitemapURL, dia.TaxonomyOptions{Limit: 0})
    if err != nil {
        return nil, fmt.Errorf("discover DIA taxonomy for import: %w", err)
    }
    return selectCategoryURLs(result.Categories, splitCSV(categoryParents), categoryLimit)
}

func selectCategoryURLs(categories []dia.TaxonomyCategory, parents []string, categoryLimit int) ([]string, error) {
    wanted := make(map[string]string, len(parents))
    for _, parent := range parents {
        normalized := strings.ToLower(strings.Trim(strings.TrimSpace(parent), "/"))
        if normalized != "" {
            wanted[normalized] = strings.TrimSpace(parent)
        }
    }
    if len(wanted) == 0 {
        return nil, fmt.Errorf("at least one non-empty --category-parents value is required")
    }

    matchedParents := make(map[string]struct{}, len(wanted))
    urls := make([]string, 0)
    for _, category := range categories {
        parent := strings.ToLower(strings.Trim(strings.TrimSpace(category.ParentPath), "/"))
        if _, ok := wanted[parent]; !ok {
            continue
        }
        url := strings.TrimSpace(category.URL)
        if url == "" {
            continue
        }
        matchedParents[parent] = struct{}{}
        urls = append(urls, url)
    }

    missing := make([]string, 0)
    for normalized, original := range wanted {
        if _, ok := matchedParents[normalized]; !ok {
            missing = append(missing, original)
        }
    }
    if len(missing) > 0 {
        sort.Strings(missing)
        return nil, fmt.Errorf("DIA taxonomy parent path(s) not found: %s", strings.Join(missing, ", "))
    }
    if len(urls) == 0 {
        return nil, fmt.Errorf("no DIA categories found for requested parent paths")
    }
    if categoryLimit > 0 && len(urls) > categoryLimit {
        return nil, fmt.Errorf("resolved %d DIA categories, exceeding --category-limit=%d; narrow --category-parents or raise the limit explicitly", len(urls), categoryLimit)
    }
    return urls, nil
}

func runDry(ctx context.Context, provider importer.Provider, postalCode string) {
    sink := &collectingSink{}
    result, err := importer.Run(ctx, provider, sink, "catalog", postalCode)
    if err != nil {
        log.Fatal(err)
    }

    output := struct {
        Mode   string            `json:"mode"`
        Result importer.Result   `json:"result"`
        Items  []catalog.Product `json:"items"`
    }{
        Mode:   "dry-run",
        Result: result,
        Items:  sink.products,
    }

    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    if err := encoder.Encode(output); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func runPersistent(ctx context.Context, provider importer.Provider, postalCode string, matchProducts bool) {
    databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
    if databaseURL == "" {
        log.Fatal("DATABASE_URL is required when --dry-run=false")
    }

    db, err := sql.Open("postgres", databaseURL)
    if err != nil {
        log.Fatalf("open postgres: %v", err)
    }
    defer db.Close()

    if err := db.PingContext(ctx); err != nil {
        log.Fatalf("ping postgres: %v", err)
    }

    persistent := postgresstore.NewSink(db)
    sink := persistentImportSink(db, persistent, matchProducts)
    result, err := importer.Run(ctx, provider, sink, "catalog", postalCode)
    if err != nil {
        log.Fatal(err)
    }

    output := struct {
        Mode          string          `json:"mode"`
        MatchProducts bool            `json:"matchProducts"`
        Result        importer.Result `json:"result"`
    }{
        Mode:          "persist",
        MatchProducts: matchProducts,
        Result:        result,
    }

    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    if err := encoder.Encode(output); err != nil {
        log.Fatal(err)
    }
}

func splitCSV(value string) []string {
    parts := strings.Split(value, ",")
    out := make([]string, 0, len(parts))
    for _, part := range parts {
        part = strings.TrimSpace(part)
        if part != "" {
            out = append(out, part)
        }
    }
    return out
}
