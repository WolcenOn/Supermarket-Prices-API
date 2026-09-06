package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/WolcenOn/Supermarket-Prices-API/internal/canonicalanalysis"
)

func main() {
	input := flag.String("input", "", "path to DIA extractor JSONL; defaults to stdin")
	output := flag.String("output", "", "path for JSON report; defaults to stdout")
	foodOnly := flag.Bool("food-only", true, "restrict analysis to known food taxonomy roots")
	minSupport := flag.Int("min-support", 2, "minimum number of products for a pattern")
	topGlobal := flag.Int("top-global", 300, "maximum global patterns")
	topPerCategory := flag.Int("top-per-category", 15, "maximum patterns per category and anchor")
	maxExamples := flag.Int("max-examples", 5, "maximum example names per pattern")
	anchors := flag.String("anchors", "jamon,pollo,tomate,leche,arroz,queso", "comma-separated terms whose local contexts should be reported")
	flag.Parse()

	reader, closeInput, err := inputReader(*input)
	if err != nil {
		fatal(err)
	}
	if closeInput != nil {
		defer closeInput()
	}

	products, err := readProducts(reader)
	if err != nil {
		fatal(err)
	}
	if len(products) == 0 {
		fatal(fmt.Errorf("no products found"))
	}

	report := canonicalanalysis.Analyze(products, canonicalanalysis.Options{
		FoodOnly:       *foodOnly,
		MinSupport:     *minSupport,
		TopGlobal:      *topGlobal,
		TopPerCategory: *topPerCategory,
		MaxExamples:    *maxExamples,
		Anchors:        splitCSV(*anchors),
	})

	writer, closeOutput, err := outputWriter(*output)
	if err != nil {
		fatal(err)
	}
	if closeOutput != nil {
		defer closeOutput()
	}
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fatal(fmt.Errorf("encode report: %w", err))
	}
}

func readProducts(reader io.Reader) ([]canonicalanalysis.Product, error) {
	scanner := bufio.NewScanner(reader)
	buffer := make([]byte, 64*1024)
	scanner.Buffer(buffer, 16*1024*1024)
	products := make([]canonicalanalysis.Product, 0)
	line := 0
	for scanner.Scan() {
		line++
		data := strings.TrimSpace(scanner.Text())
		if data == "" {
			continue
		}
		var product canonicalanalysis.Product
		if err := json.Unmarshal([]byte(data), &product); err != nil {
			return nil, fmt.Errorf("decode JSONL line %d: %w", line, err)
		}
		if strings.TrimSpace(product.Name) == "" {
			continue
		}
		products = append(products, product)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read JSONL: %w", err)
	}
	return products, nil
}

func inputReader(path string) (io.Reader, func() error, error) {
	if strings.TrimSpace(path) == "" || path == "-" {
		return os.Stdin, nil, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open input %s: %w", path, err)
	}
	return file, file.Close, nil
}

func outputWriter(path string) (io.Writer, func() error, error) {
	if strings.TrimSpace(path) == "" || path == "-" {
		return os.Stdout, nil, nil
	}
	file, err := os.Create(path)
	if err != nil {
		return nil, nil, fmt.Errorf("create output %s: %w", path, err)
	}
	return file, file.Close, nil
}

func splitCSV(value string) []string {
	var out []string
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
