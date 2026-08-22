package main

import (
    "context"
    "database/sql"
    "encoding/json"
    "flag"
    "log"
    "os"
    "strings"
    "time"

    _ "github.com/lib/pq"

    postgresstore "github.com/WolcenOn/Supermarket-Prices-API/internal/storage/postgres"
    "github.com/WolcenOn/Supermarket-Prices-API/internal/supermarkets/dia"
)

func main() {
    rawURL := flag.String("url", "", "public DIA product URL ending in /p/<sku>")
    persist := flag.Bool("persist", false, "persist sourced nutrition for an already imported product")
    timeout := flag.Duration("timeout", 30*time.Second, "maximum probe execution time")
    flag.Parse()

    if strings.TrimSpace(*rawURL) == "" {
        log.Fatal("--url is required")
    }

    ctx, cancel := context.WithTimeout(context.Background(), *timeout)
    defer cancel()

    source := dia.NewHTTPSource(nil)
    details, err := source.FetchProductDetails(ctx, strings.TrimSpace(*rawURL))
    if err != nil {
        log.Fatal(err)
    }

    if !*persist {
        writeJSON(details)
        return
    }

    databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
    if databaseURL == "" {
        log.Fatal("DATABASE_URL is required when --persist=true")
    }

    nutrition, err := details.ProductNutrition(time.Now().UTC())
    if err != nil {
        log.Fatal(err)
    }

    db, err := sql.Open("postgres", databaseURL)
    if err != nil {
        log.Fatalf("open postgres: %v", err)
    }
    defer db.Close()

    if err := db.PingContext(ctx); err != nil {
        log.Fatalf("ping postgres: %v", err)
    }

    stored, err := postgresstore.SaveProductNutrition(ctx, db, nutrition)
    if err != nil {
        log.Fatal(err)
    }

    writeJSON(struct {
        Mode   string `json:"mode"`
        Stored any    `json:"stored"`
    }{
        Mode:   "persist",
        Stored: stored,
    })
}

func writeJSON(value any) {
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    if err := encoder.Encode(value); err != nil {
        log.Fatal(err)
    }
}
