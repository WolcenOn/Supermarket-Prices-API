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

	"github.com/WolcenOn/Supermarket-Prices-API/internal/canonicalgaps"
	"github.com/WolcenOn/Supermarket-Prices-API/internal/canonicalsemantics"
	postgresstore "github.com/WolcenOn/Supermarket-Prices-API/internal/storage/postgres"
)

func main() {
	inputPath := flag.String("input", "", "canonical semantic audit JSON (required; use - for stdin)")
	outputPath := flag.String("output", "-", "output JSON path, or - for stdout")
	timeout := flag.Duration("timeout", 60*time.Second, "maximum read-only database analysis time")
	flag.Parse()

	if strings.TrimSpace(*inputPath) == "" {
		log.Fatal("--input is required")
	}
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	in, err := openInput(*inputPath)
	if err != nil {
		log.Fatal(err)
	}
	if in != os.Stdin {
		defer in.Close()
	}
	var audit canonicalsemantics.Report
	if err := json.NewDecoder(in).Decode(&audit); err != nil {
		log.Fatalf("decode semantic audit: %v", err)
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

	report, err := canonicalgaps.Analyze(ctx, postgresstore.NewCatalogStore(db), audit)
	if err != nil {
		log.Fatal(err)
	}

	out, err := openOutput(*outputPath)
	if err != nil {
		log.Fatal(err)
	}
	if out != os.Stdout {
		defer out.Close()
	}
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		log.Fatalf("encode catalog gap audit: %v", err)
	}
}

func openInput(path string) (*os.File, error) {
	if path == "-" {
		return os.Stdin, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open input %s: %w", path, err)
	}
	return file, nil
}

func openOutput(path string) (*os.File, error) {
	if path == "" || path == "-" {
		return os.Stdout, nil
	}
	file, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create output %s: %w", path, err)
	}
	return file, nil
}
