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

    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
    "github.com/WolcenOn/Supermarket-Prices-API/internal/matching"
    postgresstore "github.com/WolcenOn/Supermarket-Prices-API/internal/storage/postgres"
)

type candidate struct {
    ProductID          string         `json:"productId"`
    SupermarketID      string         `json:"supermarketId"`
    ExternalID         string         `json:"externalId"`
    ProductName        string         `json:"productName"`
    NormalizedCategory string         `json:"normalizedCategory"`
    Match              matching.Match `json:"match"`
    CanonicalExists    bool           `json:"canonicalExists"`
    Eligible           bool           `json:"eligible"`
    Reason             string         `json:"reason,omitempty"`
}

type output struct {
    Mode             string      `json:"mode"`
    Supermarket      string      `json:"supermarket"`
    Family           string      `json:"family"`
    Scanned          int         `json:"scanned"`
    Unmatched        int         `json:"unmatched"`
    CandidateCount   int         `json:"candidateCount"`
    EligibleCount    int         `json:"eligibleCount"`
    PersistedMatches int         `json:"persistedMatches"`
    Candidates       []candidate `json:"candidates"`
}

func main() {
    supermarket := flag.String("supermarket", "dia", "supermarket to inspect; current controlled phase supports dia")
    family := flag.String("family", "all", "product family: all, rice, or milk")
    limit := flag.Int("limit", 200, "maximum classified products to inspect")
    persist := flag.Bool("persist", false, "persist eligible automatic product matches; preview-only by default")
    timeout := flag.Duration("timeout", 30*time.Second, "maximum execution time")
    flag.Parse()

    supermarketID := strings.ToLower(strings.TrimSpace(*supermarket))
    familyName := strings.ToLower(strings.TrimSpace(*family))
    if supermarketID != "dia" {
        log.Fatalf("unsupported supermarket %q; current controlled phase supports dia", supermarketID)
    }
    switch familyName {
    case "all", "rice", "milk":
    default:
        log.Fatalf("unsupported family %q; expected all, rice, or milk", familyName)
    }
    if *limit <= 0 || *limit > 5000 {
        log.Fatal("--limit must be between 1 and 5000")
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

    products, err := postgresstore.LoadDeterministicMatchProducts(ctx, db, supermarketID, familyName, *limit)
    if err != nil {
        log.Fatal(err)
    }

    candidates := make([]candidate, 0)
    matchesToSave := make([]matching.Match, 0)
    canonicalExistsCache := map[string]bool{}
    unmatched := 0

    for _, product := range products {
        matches := matching.Suggest(product)
        if len(matches) == 0 {
            unmatched++
            continue
        }
        for _, match := range matches {
            exists, known := canonicalExistsCache[match.CanonicalIngredientID]
            if !known {
                exists, err = postgresstore.CanonicalIngredientExists(ctx, db, match.CanonicalIngredientID)
                if err != nil {
                    log.Fatal(err)
                }
                canonicalExistsCache[match.CanonicalIngredientID] = exists
            }
            item := candidate{
                ProductID:          product.ID,
                SupermarketID:      product.SupermarketID,
                ExternalID:         product.ExternalID,
                ProductName:        product.Name,
                NormalizedCategory: product.NormalizedCategory,
                Match:              match,
                CanonicalExists:    exists,
                Eligible:           exists,
            }
            if !exists {
                item.Reason = "missing_canonical_ingredient"
            } else {
                matchesToSave = append(matchesToSave, match)
            }
            candidates = append(candidates, item)
        }
    }

    persisted := 0
    mode := "preview"
    if *persist {
        mode = "persist"
        if err := postgresstore.SaveIngredientMatches(ctx, db, matchesToSave); err != nil {
            log.Fatal(err)
        }
        persisted = len(matchesToSave)
    }

    result := output{
        Mode:             mode,
        Supermarket:      supermarketID,
        Family:           familyName,
        Scanned:          len(products),
        Unmatched:        unmatched,
        CandidateCount:   len(candidates),
        EligibleCount:    len(matchesToSave),
        PersistedMatches: persisted,
        Candidates:       candidates,
    }
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    if err := encoder.Encode(result); err != nil {
        log.Fatal(err)
    }
}

// Keep catalog imported in the command's build graph explicit. The product
// reader returns catalog.Product and this compile-time assertion documents the
// intended boundary without introducing a second DTO for deterministic rules.
var _ = catalog.Product{}
