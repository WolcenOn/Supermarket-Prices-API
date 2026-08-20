package main

import (
    "context"
    "database/sql"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "os"
    "strings"
    "time"

    _ "github.com/lib/pq"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
    "github.com/WolcenOn/Supermarket-Prices-API/internal/importer"
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

func main() {
    supermarket := flag.String("supermarket", "dia", "supermarket provider to import")
    postalCode := flag.String("postal-code", "", "postal code used for location-sensitive observations")
    categories := flag.String("categories", "", "comma-separated DIA category URLs; defaults to a small validation set")
    dryRun := flag.Bool("dry-run", true, "fetch and normalize without persisting")
    timeout := flag.Duration("timeout", 45*time.Second, "maximum importer execution time")
    flag.Parse()

    if strings.ToLower(strings.TrimSpace(*supermarket)) != "dia" {
        log.Fatalf("unsupported supermarket %q; current phase supports dia", *supermarket)
    }

    categoryURLs := defaultDIACategories
    if strings.TrimSpace(*categories) != "" {
        categoryURLs = splitCSV(*categories)
    }

    source := dia.NewHTTPSource(categoryURLs)
    provider := dia.NewProvider(source)

    ctx, cancel := context.WithTimeout(context.Background(), *timeout)
    defer cancel()

    if *dryRun {
        runDry(ctx, provider, strings.TrimSpace(*postalCode))
        return
    }

    runPersistent(ctx, provider, strings.TrimSpace(*postalCode))
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

func runPersistent(ctx context.Context, provider importer.Provider, postalCode string) {
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

    sink := postgresstore.NewSink(db)
    result, err := importer.Run(ctx, provider, sink, "catalog", postalCode)
    if err != nil {
        log.Fatal(err)
    }

    output := struct {
        Mode   string          `json:"mode"`
        Result importer.Result `json:"result"`
    }{
        Mode:   "persist",
        Result: result,
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
