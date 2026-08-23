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
    postgresstore "github.com/WolcenOn/Supermarket-Prices-API/internal/storage/postgres"
    "github.com/WolcenOn/Supermarket-Prices-API/internal/supermarkets/dia"
)

const (
    defaultLimit = 5
    maxLimit     = 25
    minDelay     = time.Second
)

type itemResult struct {
    ProductID  string                    `json:"productId"`
    ExternalID string                    `json:"externalId"`
    Name       string                    `json:"name"`
    SourceURL  string                    `json:"sourceUrl"`
    ItemType   string                    `json:"itemType,omitempty"`
    Status     string                    `json:"status"`
    Error      string                    `json:"error,omitempty"`
    Details    *dia.ProductDetails       `json:"details,omitempty"`
    Stored     *catalog.ProductNutrition `json:"stored,omitempty"`
}

type output struct {
    Mode           string                                  `json:"mode"`
    Limit          int                                     `json:"limit"`
    Delay          string                                  `json:"delay"`
    Refresh        bool                                    `json:"refresh"`
    Diagnostics    postgresstore.DIAEnrichmentDiagnostics `json:"diagnostics"`
    Selected       int                                     `json:"selected"`
    Fetched        int                                     `json:"fetched"`
    NutritionFound int                                     `json:"nutritionFound"`
    Saved          int                                     `json:"saved"`
    Skipped        int                                     `json:"skipped"`
    Failed         int                                     `json:"failed"`
    Items          []itemResult                            `json:"items"`
}

func main() {
    limitFlag := flag.Int("limit", defaultLimit, "maximum products to inspect; hard-capped at 25")
    delayFlag := flag.Duration("delay", 2*time.Second, "pause between DIA product-page requests; minimum 1s")
    persist := flag.Bool("persist", false, "persist nutrition snapshots; preview-only by default")
    refresh := flag.Bool("refresh", false, "include products that already have DIA nutrition")
    timeout := flag.Duration("timeout", 2*time.Minute, "maximum batch execution time")
    flag.Parse()

    limit, err := boundedLimit(*limitFlag)
    if err != nil {
        log.Fatal(err)
    }
    delay := boundedDelay(*delayFlag)

    databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
    if databaseURL == "" {
        log.Fatal("DATABASE_URL is required to select already-imported DIA products")
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

    diagnostics, err := postgresstore.DIAEnrichmentSelectionDiagnostics(ctx, db)
    if err != nil {
        log.Fatal(err)
    }

    candidates, err := postgresstore.DIAEnrichmentCandidates(ctx, db, limit, *refresh)
    if err != nil {
        log.Fatal(err)
    }

    result := output{
        Mode:        mode(*persist),
        Limit:       limit,
        Delay:       delay.String(),
        Refresh:     *refresh,
        Diagnostics: diagnostics,
        Selected:    len(candidates),
        Items:       make([]itemResult, 0, len(candidates)),
    }

    source := dia.NewHTTPSource(nil)
    for index, candidate := range candidates {
        if index > 0 {
            if err := wait(ctx, delay); err != nil {
                result.Failed++
                result.Items = append(result.Items, itemResult{
                    ProductID: candidate.ProductID, ExternalID: candidate.ExternalID,
                    Name: candidate.Name, SourceURL: candidate.SourceURL, ItemType: candidate.ItemType,
                    Status: "failed", Error: err.Error(),
                })
                break
            }
        }

        item := itemResult{
            ProductID: candidate.ProductID, ExternalID: candidate.ExternalID,
            Name: candidate.Name, SourceURL: candidate.SourceURL, ItemType: candidate.ItemType,
        }

        details, err := source.FetchProductDetails(ctx, candidate.SourceURL)
        if err != nil {
            item.Status = "failed"
            item.Error = err.Error()
            result.Failed++
            result.Items = append(result.Items, item)
            continue
        }
        result.Fetched++

        if !sameExternalID(candidate.ExternalID, details.ExternalID) {
            item.Status = "failed"
            item.Error = fmt.Sprintf("DIA detail external id %q does not match stored product %q", details.ExternalID, candidate.ExternalID)
            result.Failed++
            result.Items = append(result.Items, item)
            continue
        }

        nutrition, err := details.ProductNutrition(time.Now().UTC())
        if err != nil {
            item.Status = "no_structured_nutrition"
            item.Details = &details
            result.Skipped++
            result.Items = append(result.Items, item)
            continue
        }
        result.NutritionFound++

        if !*persist {
            item.Status = "preview"
            item.Details = &details
            result.Items = append(result.Items, item)
            continue
        }

        stored, err := postgresstore.SaveProductNutrition(ctx, db, nutrition)
        if err != nil {
            item.Status = "failed"
            item.Error = err.Error()
            result.Failed++
            result.Items = append(result.Items, item)
            continue
        }
        item.Status = "saved"
        item.Stored = &stored
        result.Saved++
        result.Items = append(result.Items, item)
    }

    writeJSON(result)
}

func boundedLimit(value int) (int, error) {
    if value <= 0 {
        return 0, fmt.Errorf("--limit must be between 1 and %d", maxLimit)
    }
    if value > maxLimit {
        return maxLimit, nil
    }
    return value, nil
}

func boundedDelay(value time.Duration) time.Duration {
    if value < minDelay {
        return minDelay
    }
    return value
}

func sameExternalID(expected, actual string) bool {
    expected = strings.TrimSpace(expected)
    actual = strings.TrimSpace(actual)
    return expected != "" && expected == actual
}

func wait(ctx context.Context, delay time.Duration) error {
    timer := time.NewTimer(delay)
    defer timer.Stop()
    select {
    case <-ctx.Done():
        return ctx.Err()
    case <-timer.C:
        return nil
    }
}

func mode(persist bool) string {
    if persist {
        return "persist"
    }
    return "preview"
}

func writeJSON(value any) {
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    if err := encoder.Encode(value); err != nil {
        log.Fatal(err)
    }
}
