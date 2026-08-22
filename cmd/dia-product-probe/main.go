package main

import (
    "context"
    "encoding/json"
    "flag"
    "log"
    "os"
    "strings"
    "time"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/supermarkets/dia"
)

func main() {
    rawURL := flag.String("url", "", "public DIA product URL ending in /p/<sku>")
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

    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    if err := encoder.Encode(details); err != nil {
        log.Fatal(err)
    }
}
