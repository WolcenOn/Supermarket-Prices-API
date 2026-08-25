package main

import (
    "bytes"
    "encoding/json"
    "strings"
    "testing"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/supermarkets/dia"
)

func sampleTaxonomyResult() dia.TaxonomyResult {
    return dia.TaxonomyResult{
        SitemapURL:         "https://www.dia.es/sitemap.xml",
        DocumentsScanned:   1,
        URLEntriesScanned:  7985,
        CandidatesFound:    2,
        ExcludedNonCatalog: 0,
        Truncated:          false,
        Categories: []dia.TaxonomyCategory{
            {
                ID:         "L2107",
                Name:       "Agua",
                Path:       "agua-y-refrescos/agua/c/L2107",
                ParentPath: "agua-y-refrescos",
                URL:        "https://www.dia.es/agua-y-refrescos/agua/c/L2107",
                Depth:      2,
            },
            {
                ID:         "L2108",
                Name:       "Cola",
                Path:       "agua-y-refrescos/cola/c/L2108",
                ParentPath: "agua-y-refrescos",
                URL:        "https://www.dia.es/agua-y-refrescos/cola/c/L2108",
                Depth:      2,
            },
        },
    }
}

func TestWriteTaxonomyResultNDJSONUsesOneLinePerRecord(t *testing.T) {
    var buffer bytes.Buffer
    if err := writeTaxonomyResult(&buffer, sampleTaxonomyResult(), "ndjson"); err != nil {
        t.Fatalf("write ndjson: %v", err)
    }

    lines := strings.Split(strings.TrimSpace(buffer.String()), "\n")
    if len(lines) != 3 {
        t.Fatalf("expected summary + 2 categories, got %d lines: %q", len(lines), buffer.String())
    }

    var summary taxonomyNDJSONSummary
    if err := json.Unmarshal([]byte(lines[0]), &summary); err != nil {
        t.Fatalf("decode summary: %v", err)
    }
    if summary.Type != "summary" || summary.CategoryCount != 2 || summary.URLEntriesScanned != 7985 {
        t.Fatalf("unexpected summary: %+v", summary)
    }

    var first taxonomyNDJSONCategory
    if err := json.Unmarshal([]byte(lines[1]), &first); err != nil {
        t.Fatalf("decode first category: %v", err)
    }
    if first.Type != "category" || first.ID != "L2107" || first.Name != "Agua" {
        t.Fatalf("unexpected first category: %+v", first)
    }
}

func TestWriteTaxonomyResultKeepsPrettyJSONDefault(t *testing.T) {
    var buffer bytes.Buffer
    if err := writeTaxonomyResult(&buffer, sampleTaxonomyResult(), "json"); err != nil {
        t.Fatalf("write json: %v", err)
    }
    if !strings.Contains(buffer.String(), "\n  \"sitemapUrl\"") {
        t.Fatalf("expected indented JSON, got %q", buffer.String())
    }
}

func TestWriteTaxonomyResultRejectsUnknownFormat(t *testing.T) {
    var buffer bytes.Buffer
    err := writeTaxonomyResult(&buffer, sampleTaxonomyResult(), "csv")
    if err == nil || !strings.Contains(err.Error(), "unsupported format") {
        t.Fatalf("expected unsupported format error, got %v", err)
    }
}
