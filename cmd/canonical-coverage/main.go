package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/lib/pq"

	"github.com/WolcenOn/Supermarket-Prices-API/internal/canonicalcoverage"
	postgresstore "github.com/WolcenOn/Supermarket-Prices-API/internal/storage/postgres"
)

func main() {
	timeout := flag.Duration("timeout", 60*time.Second, "maximum execution time")
	flag.Parse()

	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	inputs, err := loadInputs(flag.Args(), os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
	if len(inputs) == 0 {
		log.Fatal("no ingredient names found; pass JSON files or JSON on stdin")
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

	report, err := canonicalcoverage.Analyze(ctx, postgresstore.NewCatalogStore(db), inputs)
	if err != nil {
		log.Fatal(err)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		log.Fatal(err)
	}
}

func loadInputs(paths []string, stdin io.Reader) ([]canonicalcoverage.Input, error) {
	if len(paths) == 0 {
		data, err := io.ReadAll(stdin)
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}
		return parseInputs(data, "stdin")
	}

	var result []canonicalcoverage.Input
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		items, err := parseInputs(data, filepath.Base(path))
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		result = append(result, items...)
	}
	return result, nil
}

func parseInputs(data []byte, source string) ([]canonicalcoverage.Input, error) {
	var stringsList []string
	if err := json.Unmarshal(data, &stringsList); err == nil {
		items := make([]canonicalcoverage.Input, 0, len(stringsList))
		for _, name := range stringsList {
			if strings.TrimSpace(name) != "" {
				items = append(items, canonicalcoverage.Input{Name: name, Count: 1, Source: source})
			}
		}
		return items, nil
	}

	var explicit []canonicalcoverage.Input
	if err := json.Unmarshal(data, &explicit); err == nil {
		for i := range explicit {
			if explicit[i].Source == "" {
				explicit[i].Source = source
			}
		}
		return explicit, nil
	}

	var document struct {
		Ingredients []struct {
			Name string `json:"name"`
		} `json:"ingredients"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("expected JSON array of names, array of {name,count,source}, or object with ingredients[]: %w", err)
	}
	if document.Ingredients == nil {
		return nil, fmt.Errorf("expected JSON array of names, array of {name,count,source}, or object with ingredients[]")
	}

	items := make([]canonicalcoverage.Input, 0, len(document.Ingredients))
	for _, ingredient := range document.Ingredients {
		if strings.TrimSpace(ingredient.Name) != "" {
			items = append(items, canonicalcoverage.Input{Name: ingredient.Name, Count: 1, Source: source})
		}
	}
	return items, nil
}
