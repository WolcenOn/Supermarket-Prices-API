package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/WolcenOn/Supermarket-Prices-API/internal/canonicalruleproposal"
	"github.com/WolcenOn/Supermarket-Prices-API/internal/canonicalsemantics"
)

func main() {
	inputPath := flag.String("input", "", "canonical rule proposal JSON (required; use - for stdin)")
	outputPath := flag.String("output", "-", "output JSON path, or - for stdout")
	flag.Parse()
	if *inputPath == "" { log.Fatal("--input is required") }

	in, err := openInput(*inputPath)
	if err != nil { log.Fatal(err) }
	if in != os.Stdin { defer in.Close() }

	var proposals canonicalruleproposal.Report
	if err := json.NewDecoder(in).Decode(&proposals); err != nil { log.Fatalf("decode proposals: %v", err) }

	out, err := openOutput(*outputPath)
	if err != nil { log.Fatal(err) }
	if out != os.Stdout { defer out.Close() }
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(canonicalsemantics.Audit(proposals)); err != nil { log.Fatalf("encode audit: %v", err) }
}

func openInput(path string) (*os.File, error) {
	if path == "-" { return os.Stdin, nil }
	file, err := os.Open(path)
	if err != nil { return nil, fmt.Errorf("open input %s: %w", path, err) }
	return file, nil
}

func openOutput(path string) (*os.File, error) {
	if path == "" || path == "-" { return os.Stdout, nil }
	file, err := os.Create(path)
	if err != nil { return nil, fmt.Errorf("create output %s: %w", path, err) }
	return file, nil
}
