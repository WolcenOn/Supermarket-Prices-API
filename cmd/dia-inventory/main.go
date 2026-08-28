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

    "github.com/WolcenOn/Supermarket-Prices-API/internal/diainventory"
)

func main() {
    postalCode := flag.String("postal-code", "", "optional postal code used to select location-specific DIA observations")
    output := flag.String("output", "summary", "output mode: summary or full")
    timeout := flag.Duration("timeout", 30*time.Second, "maximum database read time")
    flag.Parse()

    outputMode := strings.ToLower(strings.TrimSpace(*output))
    if outputMode != "summary" && outputMode != "full" {
        log.Fatal("--output must be summary or full")
    }

    databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
    if databaseURL == "" {
        log.Fatal("DATABASE_URL is required")
    }

    ctx, cancel := context.WithTimeout(context.Background(), *timeout)
    defer cancel()

    db, err := sql.Open("postgres", databaseURL)
    if err != nil {
        log.Fatalf("open postgres: %v", err)
    }
    defer db.Close()

    if err := db.PingContext(ctx); err != nil {
        log.Fatalf("ping postgres: %v", err)
    }

    report, err := diainventory.Load(ctx, db, strings.TrimSpace(*postalCode), outputMode == "full")
    if err != nil {
        log.Fatal(err)
    }

    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    if err := encoder.Encode(report); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
