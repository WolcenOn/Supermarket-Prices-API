package main

import (
    "context"
    "database/sql"
    "log"
    "net/http"
    "os"
    "strings"
    "time"

    _ "github.com/lib/pq"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/api"
    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
    postgresstore "github.com/WolcenOn/Supermarket-Prices-API/internal/storage/postgres"
)

func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    store, closeStore := buildStore()
    defer closeStore()

    handler := api.NewHandler(store)
    server := &http.Server{
        Addr:              ":" + port,
        Handler:           handler.Routes(),
        ReadHeaderTimeout: 5 * time.Second,
    }

    log.Printf("supermarket-prices-api listening on :%s", port)
    if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        log.Fatal(err)
    }
}

func buildStore() (catalog.Store, func()) {
    databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
    if databaseURL == "" {
        log.Printf("DATABASE_URL not configured; using in-memory demo catalog")
        return catalog.NewMemoryStore(catalog.SeedProducts()), func() {}
    }

    db, err := sql.Open("postgres", databaseURL)
    if err != nil {
        log.Fatalf("open postgres: %v", err)
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    if err := db.PingContext(ctx); err != nil {
        _ = db.Close()
        log.Fatalf("ping postgres: %v", err)
    }

    log.Printf("DATABASE_URL configured; serving catalog from PostgreSQL")
    return postgresstore.NewCatalogStore(db), func() { _ = db.Close() }
}
