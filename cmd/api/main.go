package main

import (
    "log"
    "net/http"
    "os"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/api"
    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    store := catalog.NewMemoryStore(catalog.SeedProducts())
    handler := api.NewHandler(store)

    server := &http.Server{
        Addr:    ":" + port,
        Handler: handler.Routes(),
    }

    log.Printf("supermarket-prices-api listening on :%s", port)
    if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        log.Fatal(err)
    }
}
