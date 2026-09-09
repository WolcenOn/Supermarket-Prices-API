package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/WolcenOn/Supermarket-Prices-API/internal/canonicalgaps"
	"github.com/WolcenOn/Supermarket-Prices-API/internal/canonicalreview"
)

func main() {
	inputPath := flag.String("input", "", "canonical catalog gap JSON (required; use - for stdin)")
	outputPath := flag.String("output", "-", "output JSON path, or - for stdout")
	flag.Parse()

	if *inputPath == "" {
		log.Fatal("--input is required")
	}

	in, err := openInput(*inputPath)
	if err != nil {
		log.Fatal(err)
	}
	if in != os.Stdin {
		defer in.Close()
	}

	var gaps canonicalgaps.Report
	if err := json.NewDecoder(in).Decode(&gaps); err != nil {
		log.Fatalf("decode canonical gap report: %v", err)
	}

	report := canonicalreview.Build(gaps)

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
		log.Fatalf("encode missing canonical review: %v", err)
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
