package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
	"github.com/WolcenOn/Supermarket-Prices-API/internal/catalogscan"
	"github.com/WolcenOn/Supermarket-Prices-API/internal/importer"
	"github.com/WolcenOn/Supermarket-Prices-API/internal/supermarkets/dia"
)

type collectingSink struct {
	products   []catalog.Product
	seen       map[string]struct{}
	duplicates int
}

func (s *collectingSink) SaveProducts(_ context.Context, products []catalog.Product) error {
	if s.seen == nil {
		s.seen = make(map[string]struct{})
	}
	for _, product := range products {
		key := strings.TrimSpace(product.SupermarketID) + "|" + strings.TrimSpace(product.ExternalID)
		if _, exists := s.seen[key]; exists && strings.TrimSpace(product.ExternalID) != "" {
			s.duplicates++
			continue
		}
		s.seen[key] = struct{}{}
		s.products = append(s.products, product)
	}
	return nil
}

type summary struct {
	Mode                 string         `json:"mode"`
	Supermarket          string         `json:"supermarket"`
	TaxonomyCategories   int            `json:"taxonomyCategories"`
	CategoriesScanned    int            `json:"categoriesScanned"`
	UniqueProducts       int            `json:"uniqueProducts"`
	DuplicateOccurrences int            `json:"duplicateOccurrences"`
	Decisions            map[string]int `json:"decisions"`
	Issues               map[string]int `json:"issues"`
	Result               importer.Result `json:"result"`
}

type fullReport struct {
	Summary summary            `json:"summary"`
	Items   []catalogscan.Item `json:"items"`
}

func main() {
	supermarket := flag.String("supermarket", "dia", "supermarket provider to scan")
	postalCode := flag.String("postal-code", "", "postal code used for location-sensitive observations")
	sitemapURL := flag.String("sitemap-url", "https://www.dia.es/sitemap.xml", "DIA sitemap URL")
	categoryParents := flag.String("category-parents", "", "comma-separated category path prefixes, e.g. verduras,huevos-leche-y-mantequilla")
	allCategories := flag.Bool("all-categories", false, "scan every discovered catalog category; must be set explicitly for a full-catalog scan")
	includeParents := flag.Bool("include-parent-categories", false, "also scan taxonomy parent categories; default scans leaf categories to reduce duplicate products")
	categoryLimit := flag.Int("category-limit", 250, "maximum selected categories; 0 disables the safety limit")
	output := flag.String("output", "summary", "output mode: summary or full")
	timeout := flag.Duration("timeout", 10*time.Minute, "maximum scan execution time")
	flag.Parse()

	if strings.ToLower(strings.TrimSpace(*supermarket)) != "dia" {
		log.Fatalf("unsupported supermarket %q; current scanner adapter supports dia", *supermarket)
	}
	if *allCategories && strings.TrimSpace(*categoryParents) != "" {
		log.Fatal("use either --all-categories or --category-parents, not both")
	}
	if !*allCategories && strings.TrimSpace(*categoryParents) == "" {
		log.Fatal("choose a scope with --category-parents or explicitly use --all-categories")
	}
	if *categoryLimit < 0 {
		log.Fatal("--category-limit must be >= 0")
	}
	outputMode := strings.ToLower(strings.TrimSpace(*output))
	if outputMode != "summary" && outputMode != "full" {
		log.Fatal("--output must be summary or full")
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	discoverer := dia.NewTaxonomyDiscoverer()
	taxonomy, err := discoverer.Discover(ctx, strings.TrimSpace(*sitemapURL), dia.TaxonomyOptions{Limit: 0})
	if err != nil {
		log.Fatalf("discover DIA taxonomy: %v", err)
	}
	if taxonomy.Truncated {
		log.Fatal("DIA taxonomy discovery was truncated; refusing to produce an incomplete catalog audit")
	}

	categories, err := selectCategories(taxonomy.Categories, splitCSV(*categoryParents), *allCategories, *includeParents, *categoryLimit)
	if err != nil {
		log.Fatal(err)
	}
	urls := make([]string, 0, len(categories))
	for _, category := range categories {
		urls = append(urls, category.URL)
	}
	log.Printf("catalog scan is read-only: scanning %d DIA categories; no database writes will be performed", len(urls))

	source := dia.NewHTTPSource(urls)
	provider := dia.NewProvider(source)
	sink := &collectingSink{}
	result, err := importer.Run(ctx, provider, sink, "catalog-scan", strings.TrimSpace(*postalCode))
	if err != nil {
		log.Fatal(err)
	}

	items := catalogscan.AnalyzeAll(sink.products)
	reportSummary := buildSummary(taxonomy, categories, sink, result, items)

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if outputMode == "summary" {
		if err := encoder.Encode(reportSummary); err != nil {
			log.Fatal(err)
		}
		return
	}
	if err := encoder.Encode(fullReport{Summary: reportSummary, Items: items}); err != nil {
		log.Fatal(err)
	}
}

func selectCategories(categories []dia.TaxonomyCategory, prefixes []string, all, includeParents bool, limit int) ([]dia.TaxonomyCategory, error) {
	leaf := leafCategoryIDs(categories)
	wanted := normalizePrefixes(prefixes)
	matched := make(map[string]struct{})
	selected := make([]dia.TaxonomyCategory, 0)

	for _, category := range categories {
		if !includeParents {
			if _, ok := leaf[strings.ToLower(strings.TrimSpace(category.ID))]; !ok {
				continue
			}
		}
		if !all {
			path := strings.ToLower(strings.Trim(strings.TrimSpace(category.Path), "/"))
			include := false
			for _, prefix := range wanted {
				if path == prefix || strings.HasPrefix(path, prefix+"/") {
					include = true
					matched[prefix] = struct{}{}
				}
			}
			if !include {
				continue
			}
		}
		if strings.TrimSpace(category.URL) != "" {
			selected = append(selected, category)
		}
	}

	if !all {
		missing := make([]string, 0)
		for _, prefix := range wanted {
			if _, ok := matched[prefix]; !ok {
				missing = append(missing, prefix)
			}
		}
		if len(missing) > 0 {
			sort.Strings(missing)
			return nil, fmt.Errorf("DIA category path prefix(es) not found: %s", strings.Join(missing, ", "))
		}
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("no DIA categories selected")
	}
	if limit > 0 && len(selected) > limit {
		return nil, fmt.Errorf("selected %d DIA categories, exceeding --category-limit=%d; narrow the scope or raise the limit explicitly", len(selected), limit)
	}
	return selected, nil
}

func leafCategoryIDs(categories []dia.TaxonomyCategory) map[string]struct{} {
	parentPaths := make(map[string]struct{})
	for _, category := range categories {
		parent := strings.ToLower(strings.Trim(strings.TrimSpace(category.ParentPath), "/"))
		if parent != "" {
			parentPaths[parent] = struct{}{}
		}
	}
	leaf := make(map[string]struct{})
	for _, category := range categories {
		path := strings.ToLower(strings.Trim(strings.TrimSpace(category.Path), "/"))
		catalogPath := path
		if index := strings.Index(catalogPath, "/c/"); index >= 0 {
			catalogPath = catalogPath[:index]
		}
		if _, isParent := parentPaths[catalogPath]; !isParent {
			leaf[strings.ToLower(strings.TrimSpace(category.ID))] = struct{}{}
		}
	}
	return leaf
}

func normalizePrefixes(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.Trim(strings.TrimSpace(value), "/"))
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func buildSummary(taxonomy dia.TaxonomyResult, categories []dia.TaxonomyCategory, sink *collectingSink, result importer.Result, items []catalogscan.Item) summary {
	out := summary{
		Mode:                 "read-only-audit",
		Supermarket:          "dia",
		TaxonomyCategories:   len(taxonomy.Categories),
		CategoriesScanned:    len(categories),
		UniqueProducts:       len(sink.products),
		DuplicateOccurrences: sink.duplicates,
		Decisions:            make(map[string]int),
		Issues:               make(map[string]int),
		Result:               result,
	}
	for _, item := range items {
		out.Decisions[item.Analysis.Decision]++
		for _, issue := range item.Analysis.Issues {
			out.Issues[issue.Code]++
		}
	}
	return out
}
