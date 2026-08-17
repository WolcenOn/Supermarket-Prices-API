package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "os"
    "strings"
    "time"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/importer"
    "github.com/WolcenOn/Supermarket-Prices-API/internal/supermarkets/dia"
)

var defaultDIACategories = []string{
    "https://www.dia.es/arroz-pastas-y-legumbres/arroz/c/L2042?page=1",
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

    result, err := importer.Run(ctx, provider, nil, "catalog", strings.TrimSpace(*postalCode))
    if err != nil {
        log.Fatal(err)
    }

    if !*dryRun {
        log.Fatal("persistence is not wired yet; run with --dry-run or implement the PostgreSQL Sink")
    }

    products, err := provider.Search(ctx, "catalog", strings.TrimSpace(*postalCode))
    if err != nil {
        log.Fatal(err)
    }

    output := struct {
        Mode   string      `json:"mode"`
        Result interface{} `json:"result"`
        Items  interface{} `json:"items"`
    }{
        Mode:   "dry-run",
        Result: result,
        Items:  products,
    }

    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    if err := encoder.Encode(output); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
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
